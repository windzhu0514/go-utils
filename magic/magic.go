package magic

// incomparable is a zero-width, non-comparable type. Adding it to a struct
// makes that struct also non-comparable, and generally doesn't add
// any size (as long as it's first).
type incomparable [0]func()

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}

// lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) lock()   {}
func (*noCopy) unlock() {}

// align64 may be added to structs that must be 64-bit aligned.
// This struct is recognized by a special case in the compiler
// and will not work if copied to any other package.
type align64 struct{}
