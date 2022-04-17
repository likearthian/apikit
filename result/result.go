package result

type Result[T any] interface {
	IsOk() bool
	IsErr() bool
	Map(func(T)) Result[T]
	MapErr(func(error)) Result[T]
}

func MapResult[T any, U any](res Result[T], mapFn func(T) U) Result[U] {
	var newRes Result[U]
	res.
		Map(func(val T) {
			newRes = Ok(mapFn(val))
		}).
		MapErr(func(err error) {
			newRes = Err[U](err)
		})

	return newRes
}

func UnwrapOrElse[T any](res Result[T], elseVal T) T {
	var payload = elseVal
	res.
		Map(func(val T) {
			payload = val
		})

	return payload
}

type result[T any] struct {
	val T
	err error
}

func (r result[T]) IsErr() bool {
	return r.err != nil
}

func (r result[T]) IsOk() bool {
	return !r.IsErr()
}

func (r result[T]) Map(fn func(val T)) Result[T] {
	if r.IsOk() {
		fn(r.val)
	}

	return r
}

func (r result[T]) MapErr(fn func(err error)) Result[T] {
	if r.IsErr() {
		fn(r.err)
	}

	return r
}

func Ok[T any](val T) Result[T] {
	return result[T]{
		val: val,
	}
}

func Err[T any](err error) Result[T] {
	return result[T]{
		err: err,
	}
}
