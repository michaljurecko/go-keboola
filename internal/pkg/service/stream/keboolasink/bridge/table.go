package bridge

import (
	"context"
	"reflect"

	"github.com/keboola/go-client/pkg/keboola"
	"go.opentelemetry.io/otel/attribute"

	"github.com/keboola/keboola-as-code/internal/pkg/encoding/json"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/ctxattr"
	serviceError "github.com/keboola/keboola-as-code/internal/pkg/service/common/errors"
	"github.com/keboola/keboola-as-code/internal/pkg/service/common/rollback"
	"github.com/keboola/keboola-as-code/internal/pkg/service/stream/definition"
	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

func (b *Bridge) ensureTableExists(ctx context.Context, tableKey keboola.TableKey, sink definition.Sink) error {
	// Create bucket if not exists
	if err := b.ensureBucketExists(ctx, tableKey.BucketKey()); err != nil {
		return err
	}

	// Get table
	tab, err := b.getTable(ctx, tableKey)

	// Create table
	columnsNames := sink.Table.Mapping.Columns.Names()
	primaryKey := sink.Table.Mapping.Columns.PrimaryKey()
	var apiErr *keboola.StorageError
	if errors.As(err, &apiErr) && apiErr.ErrCode == "storage.tables.notFound" {
		// Create table
		tab, err = b.createTable(ctx, tableKey, columnsNames, primaryKey)
	}

	// Handle get/create error
	if err != nil {
		return err
	}

	// add table metadata
	if err = b.addTableMetadata(ctx, tab, sink); err != nil {
		return err
	}

	// Check columns
	if !reflect.DeepEqual(columnsNames, tab.Columns) {
		return serviceError.NewBadRequestError(errors.Errorf(
			`columns of the table "%s" do not match expected %s, found %s`,
			tab.TableID.String(), json.MustEncodeString(columnsNames, false), json.MustEncodeString(tab.Columns, false),
		))
	}
	// Check primary key
	if !reflect.DeepEqual(primaryKey, tab.PrimaryKey) {
		return serviceError.NewBadRequestError(errors.Errorf(
			`primary key of the table "%s" does not match expected %s, found %s`,
			tab.TableID.String(), json.MustEncodeString(primaryKey, false), json.MustEncodeString(tab.PrimaryKey, false),
		))
	}

	return nil
}

func (b *Bridge) getTable(ctx context.Context, tableKey keboola.TableKey) (*keboola.Table, error) {
	api, err := b.apiProvider.APIFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return api.GetTableRequest(tableKey).Send(ctx)
}

func (b *Bridge) createTable(ctx context.Context, tableKey keboola.TableKey, columnNames, primaryKey []string) (*keboola.Table, error) {
	ctx = ctxattr.ContextWith(
		ctx,
		attribute.String("table.key", tableKey.String()),
		attribute.StringSlice("table.columns", columnNames),
		attribute.StringSlice("table.pk", primaryKey),
	)

	rb := rollback.FromContext(ctx)
	api, err := b.apiProvider.APIFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Create table definition
	tableDef := keboola.TableDefinition{PrimaryKeyNames: primaryKey}
	for _, name := range columnNames {
		tableDef.Columns = append(tableDef.Columns, keboola.Column{Name: name})
	}

	// Create table
	b.logger.Info(ctx, "creating table")
	tab, err := api.CreateTableDefinitionRequest(tableKey, tableDef).Send(ctx)
	if err != nil {
		return nil, err
	}

	// Register rollback
	rb.Add(func(ctx context.Context) error {
		ctx = ctxattr.ContextWith(ctx, attribute.String("table.key", tableKey.String()))
		b.logger.Info(ctx, "rollback: deleting table")
		return api.DeleteTableRequest(tableKey).SendOrErr(ctx)
	})

	b.logger.Info(ctx, "created table")
	return tab, nil
}

func (b *Bridge) addTableMetadata(ctx context.Context, table *keboola.Table, sink definition.Sink) error {
	api, err := b.apiProvider.APIFromContext(ctx)
	if err != nil {
		return err
	}

	for i := range table.Metadata {
		if table.Metadata[i].Key == exportMetaKey {
			return nil
		}
	}

	return api.CreateOrUpdateTableMetadata(
		table.TableKey,
		"stream",
		[]keboola.TableMetadataRequest{
			{Key: exportMetaKey, Value: sink.SinkID.String()},
		},
		[]keboola.ColumnMetadataRequest{},
	).SendOrErr(ctx)
}
