package core

import "io"

// *
// 아직 왜 선언한지 모르겠음
type Encoder[T any] interface {
	Encode(io.Writer, T) error
}

// *
type Decoder[T any] interface {
	Decode(io.Reader, T) error
}
