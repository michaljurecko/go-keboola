package transport

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/yamux"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/logger"
	"go.opentelemetry.io/otel/attribute"

	"github.com/keboola/keboola-as-code/internal/pkg/service/common/ctxattr"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

type ClientConnection struct {
	client     *Client
	targetAddr string

	lastError error

	lock sync.Mutex
	sess *yamux.Session
}

func newClientConnection(remoteAddr string, c *Client) (*ClientConnection, error) {
	// Stop, if the client is closed
	if c.isClosed() {
		return nil, yamux.ErrSessionShutdown
	}

	ctx := ctxattr.ContextWith(
		context.Background(),
		attribute.String("nodeId", c.nodeID),
		attribute.String("targetAddress", remoteAddr),
	)

	conn := &ClientConnection{
		client:     c,
		targetAddr: remoteAddr,
	}

	// Dial connection
	initDone := make(chan error, 1)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		conn.dialLoop(ctx, remoteAddr, initDone)
	}()

	// Wait for the first connect attempt
	if err := <-initDone; err != nil {
		return nil, err
	}

	sess, err := conn.session()
	if err != nil {
		return nil, err
	}

	c.logger.Infof(ctx, `disk writer client connected from %q to %q`, sess.LocalAddr().String(), remoteAddr)
	return conn, nil
}

func (c *ClientConnection) OpenStream() (*ClientStream, error) {
	// Stop, if the client is closed
	if c.client.isClosed() {
		return nil, yamux.ErrSessionShutdown
	}

	// Get session, if connected
	sess, err := c.session()
	if err != nil {
		return nil, err
	}

	// Open stream and wrap the stream Close method
	stream, err := sess.OpenStream()
	if err != nil {
		return nil, err
	}
	return newClientStream(stream, c.client), nil
}

func (c *ClientConnection) Close() (err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if sess := c.sess; sess != nil {
		err = sess.Close()
		c.sess = nil
		c.lastError = nil
		c.client.unregisterSession(sess)
	}

	return err
}

func (c *ClientConnection) dialLoop(ctx context.Context, targetAddr string, initDone chan error) {
	// Close connection session on shutdown
	defer func() {
		if err := c.Close(); err != nil {
			err := errors.PrefixError(err, `cannot close connection`)
			logger.Error(ctx, err.Error())
		}
	}()

	b := newClientConnBackoff()

	for {
		if c.client.isClosed() {
			return
		}

		// Create session
		sess, err := c.newSession(targetAddr)

		// Finish initialization after the first connection attempt
		if initDone != nil {
			if err != nil {
				initDone <- err
				close(initDone)
				return
			}
			close(initDone)
			initDone = nil
		}

		// Handle connection error with backoff
		if err != nil {
			delay := b.NextBackOff()
			c.client.logger.Errorf(ctx, `%s, waiting %s before retry`, err, delay)
			<-time.After(delay)
			continue
		}

		b.Reset()

		// Block while the connection is open
		<-sess.CloseChan()
	}
}

func (c *ClientConnection) newSession(targetAddr string) (sess *yamux.Session, err error) {
	defer func() {
		// Update internal state
		c.lock.Lock()
		defer c.lock.Unlock()
		c.sess = sess
		c.lastError = err

		if sess != nil {
			c.client.registerSession(sess)
		}
	}()

	// Create connection
	conn, err := c.client.transport.Dial(targetAddr)
	if err != nil {
		return nil, errors.PrefixError(err, "cannot dial connection")
	}

	// Create multiplexer
	sess, err = yamux.Client(conn, multiplexerConfig(c.client.logger, c.client.config))
	if err != nil {
		_ = conn.Close()
		return nil, errors.PrefixError(err, "cannot create client multiplexer")
	}

	return sess, nil
}

func (c *ClientConnection) session() (*yamux.Session, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.sess == nil {
		if c.lastError != nil {
			return nil, errors.PrefixError(c.lastError, "cannot open stream: no connection")
		}
		return nil, errors.New("cannot open stream: no connection")
	}

	return c.sess, nil
}

func newClientConnBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.RandomizationFactor = 0.2
	b.Multiplier = 2
	b.InitialInterval = 5 * time.Millisecond
	b.MaxInterval = 5 * time.Second
	b.MaxElapsedTime = 0 // don't stop
	b.Reset()
	return b
}
