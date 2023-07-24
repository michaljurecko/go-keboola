package etcdop

import (
	"context"

	etcd "go.etcd.io/etcd/client/v3"

	"github.com/keboola/keboola-as-code/internal/pkg/log"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/etcdop/op"
)

type WatchEventT[T any] struct {
	Type   EventType
	Kv     *op.KeyValue
	PrevKv *op.KeyValue
	Value  T
	// PrevValue is set only for UpdateEvent if clientv3.WithPrevKV() option is used.
	PrevValue *T
}

type WatchStreamT[T any] WatchStreamE[WatchEventT[T]]

func (s *WatchStreamT[T]) Channel() <-chan WatchResponseE[WatchEventT[T]] {
	return s.channel
}

func (s *WatchStreamT[T]) SetupConsumer(logger log.Logger) WatchConsumer[WatchEventT[T]] {
	stream := WatchStreamE[WatchEventT[T]](*s)
	return newConsumer[WatchEventT[T]](logger, &stream)
}

// GetAllAndWatch loads all keys in the prefix by the iterator and then watch for changes.
// Values are decoded to the type T.
//
// If a fatal error occurs, the watcher is restarted.
// The "restarted" event is emitted before the restart.
// Then, the following events are streamed from the beginning.
//
// See WatchResponse for details.
func (v PrefixT[T]) GetAllAndWatch(ctx context.Context, client *etcd.Client, opts ...etcd.OpOption) (out *WatchStreamT[T]) {
	return v.decodeChannel(ctx, func(ctx context.Context) *WatchStream {
		return v.prefix.GetAllAndWatch(ctx, client, opts...)
	})
}

// Watch method wraps low-level etcd watcher.
// Values are decoded to the type T.
//
// In addition, if a fatal error occurs, the watcher is restarted.
// The "restarted" event is emitted before the restart.
// Then, the following events are streamed from the beginning.
//
// If the InitErr occurs during the first attempt to create the watcher,
// the operation is stopped and the restart is not performed.
//
// See WatchResponse for details.
func (v PrefixT[T]) Watch(ctx context.Context, client etcd.Watcher, opts ...etcd.OpOption) *WatchStreamT[T] {
	return v.decodeChannel(ctx, func(ctx context.Context) *WatchStream {
		return v.prefix.Watch(ctx, client, opts...)
	})
}

// decodeChannel is used by Watch and GetAllAndWatch to decode raw data to typed data.
func (v PrefixT[T]) decodeChannel(ctx context.Context, channelFactory func(ctx context.Context) *WatchStream) *WatchStreamT[T] {
	stream := &WatchStreamT[T]{channel: make(chan WatchResponseE[WatchEventT[T]])}
	go func() {
		defer close(stream.channel)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Decode value, if an error occurs, send it through the channel.
		decode := func(kv *op.KeyValue, header *Header) (T, bool) {
			var target T
			if err := v.serde.Decode(ctx, kv, &target); err != nil {
				resp := WatchResponseE[WatchEventT[T]]{}
				resp.Header = header
				resp.Err = err
				stream.channel <- resp
				return target, false
			}
			return target, true
		}

		// Channel is closed by the context, so the context does not have to be checked here again.
		rawStream := channelFactory(ctx)
		for rawResp := range rawStream.channel {
			var events []WatchEventT[T]
			if len(rawResp.Events) > 0 {
				events = make([]WatchEventT[T], 0, len(rawResp.Events))
			}

			// Map raw response to typed response.
			for _, rawEvent := range rawResp.Events {
				outEvent := WatchEventT[T]{
					Type:   rawEvent.Type,
					Kv:     rawEvent.Kv,
					PrevKv: rawEvent.PrevKv,
				}

				// Decode value.
				var ok bool
				if rawEvent.Type == CreateEvent || rawEvent.Type == UpdateEvent {
					// Always decode create/update value.
					if outEvent.Value, ok = decode(rawEvent.Kv, rawResp.Header); !ok {
						continue
					}
				} else if rawEvent.Type == DeleteEvent && rawEvent.PrevKv != nil {
					// Decode previous value on delete, if is present.
					// etcd.WithPrevKV() option must be used to enable it.
					if outEvent.Value, ok = decode(rawEvent.PrevKv, rawResp.Header); !ok {
						continue
					}
				}

				// Decode previous value on update, if it is present.
				// etcd.WithPrevKV() option must be used to enable it.
				if rawEvent.Type == UpdateEvent && rawEvent.PrevKv != nil {
					if prevValue, ok := decode(rawEvent.PrevKv, rawResp.Header); ok {
						outEvent.PrevValue = &prevValue
					} else {
						continue
					}
				}

				events = append(events, outEvent)
			}

			// Pass the response
			stream.channel <- WatchResponseE[WatchEventT[T]]{
				WatcherStatus: rawResp.WatcherStatus,
				Events:        events,
			}
		}
	}()

	return stream
}
