package lbengine

import (
	"crypto/sha3"
	"fmt"
	"hash"
	"io"
	"slices"

	"github.com/leafbridge/leafbridge/core/filehash"
	"github.com/leafbridge/leafbridge/core/lbdeploy"
)

// FileVerifier is capable of absorbing file content as a file is read or
// downloaded. When finished, it can produce a set of attributes for the file,
// including its cryptographic hash sums.
type FileVerifier struct {
	size   int64
	hashes map[filehash.Type]hash.Hash
}

// NewFileVerifier returns a file verifier that will generate the provided
// file hash types.
//
// It returns an error if any of the file hash types are not recognized.
func NewFileVerifier(hashTypes ...filehash.Type) (*FileVerifier, error) {
	v := FileVerifier{
		hashes: make(map[filehash.Type]hash.Hash, len(hashTypes)),
	}
	for _, typ := range hashTypes {
		if _, exists := v.hashes[typ]; exists {
			continue
		}
		switch typ {
		case filehash.SHA3_256:
			v.hashes[typ] = sha3.New256()
		default:
			return nil, fmt.Errorf("unrecognized file hash type \"%s\"", typ)
		}
	}
	return &v, nil
}

// Size returns the number of bytes that have been written to the verifier so
// far.
func (v *FileVerifier) Size() int64 {
	return v.size
}

// HashTypes returns an ordered list of hash types that the verifier is
// producing.
func (v *FileVerifier) HashTypes() []filehash.Type {
	types := make([]filehash.Type, 0, len(v.hashes))
	for typ := range v.hashes {
		types = append(types, typ)
	}
	slices.SortFunc(types, filehash.CompareTypes)
	return types
}

// Write absorbs more file data into the file verifier's state.
func (v *FileVerifier) Write(p []byte) (n int, err error) {
	v.size += int64(len(p))
	for t, hash := range v.hashes {
		if _, err := hash.Write(p); err != nil {
			return 0, fmt.Errorf("%s: %w", t, err)
		}
	}
	return len(p), nil
}

// ReadFrom reads data from r until it encounters io.EOF or an error.
//
// It returns the total number of bytes read.
func (v *FileVerifier) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [262144]byte // 256 KB
	for {
		chunk, err := r.Read(buf[:])
		if chunk > 0 {
			n += int64(chunk)
			if _, err := v.Write(buf[:chunk]); err != nil {
				return n, err
			}
		}
		if err != nil {
			if err == io.EOF {
				return n, nil
			}
			return n, err
		}
	}
}

// Reset resets the verifier to its initial state.
func (v *FileVerifier) Reset() {
	v.size = 0
	for _, hash := range v.hashes {
		hash.Reset()
	}
}

// State returns the current attributes of the file being verified.
func (v *FileVerifier) State() lbdeploy.FileAttributes {
	attrs := lbdeploy.FileAttributes{
		Size: v.size,
	}
	if len(v.hashes) > 0 {
		attrs.Hashes = make(filehash.Map, len(v.hashes))
		for t, hash := range v.hashes {
			attrs.Hashes[t] = hash.Sum(nil)
		}
	}
	return attrs
}
