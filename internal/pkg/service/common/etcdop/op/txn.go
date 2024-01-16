package op

import (
	"context"

	etcd "go.etcd.io/etcd/client/v3"

	"github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

// TxnOp provides a high-level interface built on top of etcd.Txn., see If, Then and Else methods.
//
// For more information on etcd transactions, please refer to:
// https://etcd.io/docs/v3.5/learning/api/#transaction
//
// Like other operations in this package, you can define processors using the AddProcessor method.
//
// High-level TxnOp is composed of high-level Op operations.
// This allows you to easily combine operations into an atomic transaction
// Processors defined in the operations will be executed.
//
// Another advantage is the ability to combine several TxnOp transactions into one, see Add method.
//
// R is type of the transaction result, use NoResult type if you don't need it.
// The results of individual sub-operations can be obtained using TxnResult.SubResults.
type TxnOp[R any] struct {
	result     *R
	client     etcd.KV
	processors []func(ctx context.Context, r *TxnResult[R])
	ifs        []etcd.Cmp
	thenOps    []Op
	elseOps    []Op
	andOps     []Op
}

type lowLevelTxn[R any] struct {
	result      *R
	client      etcd.KV
	processors  []func(ctx context.Context, r *TxnResult[R])
	ifs         []etcd.Cmp
	thenOps     []etcd.Op
	elseOps     []etcd.Op
	thenMappers []MapFn
	elseMappers []MapFn
}

// Txn creates an empty transaction with NoResult.
func Txn(client etcd.KV) *TxnOp[NoResult] {
	return &TxnOp[NoResult]{client: client, result: &NoResult{}}
}

// TxnWithResult creates an empty transaction with the result.
func TxnWithResult[R any](client etcd.KV, result *R) *TxnOp[R] {
	return &TxnOp[R]{client: client, result: result}
}

// MergeToTxn merges listed operations into a transaction using And method.
func MergeToTxn(client etcd.KV, ops ...Op) *TxnOp[NoResult] {
	return Txn(client).And(ops...)
}

// Then takes a list of operations.
// The operations will be executed, if the comparisons passed in If() succeed.
func (v *TxnOp[R]) Then(ops ...Op) *TxnOp[R] {
	v.thenOps = append(v.thenOps, ops...)
	return v
}

func (v *TxnOp[R]) Empty() bool {
	return len(v.ifs) == 0 && len(v.thenOps) == 0 && len(v.elseOps) == 0 && len(v.andOps) == 0
}

// If takes a list of comparison.
// If all comparisons passed in succeed, the operations passed into Then() will be executed,
// otherwise the operations passed into Else() will be executed.
func (v *TxnOp[R]) If(cs ...etcd.Cmp) *TxnOp[R] {
	v.ifs = append(v.ifs, cs...)
	return v
}

// Else takes a list of operations.
// The operations list will be executed, if any from comparisons passed in If() fail.
func (v *TxnOp[R]) Else(ops ...Op) *TxnOp[R] {
	v.elseOps = append(v.elseOps, ops...)
	return v
}

// And merges the transaction with one or more other transactions.
// IF conditions from all transactions are merged and must be fulfilled, to invoke the Then branch,
// otherwise the Else branch is executed.
// The processor from all transactions are preserved and executed.
// For non-transactions operations, the method behaves same as the Then.
func (v *TxnOp[R]) And(ops ...Op) *TxnOp[R] {
	v.andOps = append(v.andOps, ops...)
	return v
}

// AddProcessor adds a processor callback which is always executed after the transaction.
func (v *TxnOp[R]) AddProcessor(p func(ctx context.Context, r *TxnResult[R])) *TxnOp[R] {
	v.processors = append(v.processors, p)
	return v
}

// OnResult is a shortcut for the AddProcessor.
// If no error occurred yet, then the callback is executed with the result.
func (v *TxnOp[R]) OnResult(fn func(result *TxnResult[R])) *TxnOp[R] {
	return v.AddProcessor(func(_ context.Context, r *TxnResult[R]) {
		if r.Err() == nil {
			fn(r)
		}
	})
}

// OnSucceeded is a shortcut for the AddProcessor.
// If no error occurred yet and the transaction is succeeded, then the callback is executed.
func (v *TxnOp[R]) OnSucceeded(fn func(result *TxnResult[R])) *TxnOp[R] {
	return v.AddProcessor(func(_ context.Context, r *TxnResult[R]) {
		if r.Err() == nil && r.Succeeded() {
			fn(r)
		}
	})
}

func (v *TxnOp[R]) Do(ctx context.Context, opts ...Option) *TxnResult[R] {
	if lowLevel, err := v.lowLevelTxn(ctx); err == nil {
		return lowLevel.Do(ctx, opts...)
	} else {
		return newErrorTxnResult[R](err)
	}
}

func (v *TxnOp[R]) Op(ctx context.Context) (LowLevelOp, error) {
	if lowLevel, err := v.lowLevelTxn(ctx); err == nil {
		return lowLevel.Op(ctx)
	} else {
		return LowLevelOp{}, err
	}
}

func (v *TxnOp[R]) lowLevelTxn(ctx context.Context) (*lowLevelTxn[R], error) {
	out := &lowLevelTxn[R]{result: v.result, client: v.client, thenOps: make([]etcd.Op, 0), elseOps: make([]etcd.Op, 0)}
	errs := errors.NewMultiError()

	// Copy processors
	out.processors = make([]func(ctx context.Context, r *TxnResult[R]), len(v.processors))
	copy(out.processors, v.processors)

	// Copy IFs
	out.ifs = make([]etcd.Cmp, len(v.ifs))
	copy(out.ifs, v.ifs)

	// Map THEN operations
	for i, op := range v.thenOps {
		// Create low-level operation
		if lowLevel, err := op.Op(ctx); err == nil {
			out.addThen(lowLevel.Op, lowLevel.MapResponse)
		} else {
			errs.Append(errors.PrefixErrorf(err, "cannot create operation [then][%d]", i))
		}
	}

	// Map ELSE operations
	for i, op := range v.elseOps {
		// Create low-level operation
		if lowLevel, err := op.Op(ctx); err == nil {
			out.addElse(lowLevel.Op, lowLevel.MapResponse)
		} else {
			errs.Append(errors.PrefixErrorf(err, "cannot create operation [else][%d]", i))
		}
	}

	// Map AND operations, merge transactions
	for i, op := range v.andOps {
		// Create low-level operation
		lowLevel, err := op.Op(ctx)
		if err != nil {
			errs.Append(errors.PrefixErrorf(err, "cannot create operation [and][%d]", i))
			continue
		}

		// If it is not a transaction, process it as Then
		if !lowLevel.Op.IsTxn() {
			out.addThen(lowLevel.Op, lowLevel.MapResponse)
			continue
		}

		// Get transaction parts
		ifs, thenOps, elseOps := lowLevel.Op.Txn()

		// Merge IFs
		out.ifs = append(out.ifs, ifs...)

		// Merge THEN operations
		// The THEN branch will be applied, if all conditions (from all sub-transactions) are met.
		thenStart := len(out.thenOps)
		thenEnd := thenStart + len(thenOps)
		for _, item := range thenOps {
			out.addThen(item, nil)
		}

		// Merge ELSE operations
		// The ELSE branch will be applied only if the conditions of the sub-transaction are not met
		elsePos := -1
		if len(elseOps) > 0 || len(ifs) > 0 {
			elsePos = out.addElse(etcd.OpTxn(ifs, []etcd.Op{}, elseOps), nil)
		}

		// There may be a situation where neither THEN nor ELSE branch is executed:
		// If the transaction fails, but the reason is not in this sub-transaction.

		// On result, compose and map response that corresponds to the original sub-transaction
		out.processors = append(out.processors, func(ctx context.Context, r *TxnResult[R]) {
			// Get sub-transaction response
			var subTxnResponse *etcd.TxnResponse
			switch {
			case r.succeeded:
				subTxnResponse = &etcd.TxnResponse{
					// The entire transaction succeeded, which means that a partial transaction succeeded as well
					Succeeded: true,
					// Compose responses that corresponds to the original sub-transaction
					Responses: r.Response().Txn().Responses[thenStart:thenEnd],
				}
			case elsePos >= 0:
				subTxnResponse = (*etcd.TxnResponse)(r.Response().Txn().Responses[elsePos].GetResponseTxn())
				if subTxnResponse.Succeeded {
					// Skip mapper bellow, the transaction failed, but not due to a condition in the sub-transaction
					r.AddSubResult(NoResult{})
					return
				}
			default:
				// Skip mapper bellow, the transaction failed, but there is no condition in the sub-transaction
				r.AddSubResult(NoResult{})
				return
			}

			// Call original mapper of the sub transaction
			if subResult, err := lowLevel.MapResponse(ctx, r.Response().SubResponse(subTxnResponse.OpResponse())); err == nil {
				r.AddSubResult(subResult)
			} else {
				r.AddSubResult(err).AddErr(err)
			}
		})
	}

	if err := errs.ErrorOrNil(); err != nil {
		return nil, err
	}

	return out, nil
}

func (v *lowLevelTxn[R]) Op(_ context.Context) (LowLevelOp, error) {
	return v.op(), nil
}

func (v *lowLevelTxn[R]) op() LowLevelOp {
	return LowLevelOp{
		Op: etcd.OpTxn(v.ifs, v.thenOps, v.elseOps),
		MapResponse: func(ctx context.Context, raw RawResponse) (result any, err error) {
			txnResult := v.mapResponse(ctx, raw)
			return txnResult, txnResult.Err()
		},
	}
}

func (v *lowLevelTxn[R]) Do(ctx context.Context, opts ...Option) *TxnResult[R] {
	// Create low-level operation
	op := v.op()

	// Do with retry
	response, err := DoWithRetry(ctx, v.client, op.Op, opts...)
	if err != nil {
		return newErrorTxnResult[R](err)
	}

	return v.mapResponse(ctx, response)
}

func (v *lowLevelTxn[R]) addThen(op etcd.Op, mapper MapFn) {
	v.thenOps = append(v.thenOps, op)
	v.thenMappers = append(v.thenMappers, mapper)
}

func (v *lowLevelTxn[R]) addElse(op etcd.Op, mapper MapFn) (index int) {
	index = len(v.elseOps)
	v.elseOps = append(v.elseOps, op)
	v.elseMappers = append(v.elseMappers, mapper)
	return index
}

func (v *lowLevelTxn[R]) mapResponse(ctx context.Context, raw RawResponse) *TxnResult[R] {
	// Map transaction response
	r := newTxnResult(&raw, v.result)
	r.succeeded = raw.Txn().Succeeded

	// Map sub-responses
	for i, subResponse := range raw.Txn().Responses {
		// Get mapper
		var mapper MapFn
		if r.succeeded {
			mapper = v.thenMappers[i]
		} else {
			mapper = v.elseMappers[i]
		}

		// Use mapper
		if mapper != nil {
			if subResult, err := mapper(ctx, raw.SubResponse(mapRawResponse(subResponse))); err == nil {
				r.AddSubResult(subResult)
			} else {
				r.AddErr(err)
			}
		}
	}

	// Use processors
	for _, p := range v.processors {
		p(ctx, r)
	}

	return r
}
