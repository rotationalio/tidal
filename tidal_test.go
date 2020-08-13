package tidal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	defer Reset()
	// Register migrations out of order and ensure they are in sorted order
	require.NoError(t, Register(Migration{Revision: 23}))
	require.NoError(t, Register(Migration{Revision: 2}))
	require.NoError(t, Register(Migration{Revision: 9}))
	require.NoError(t, Register(Migration{Revision: 8}))
	require.NoError(t, Register(Migration{Revision: 41}))
	require.NoError(t, Register(Migration{Revision: 5}))
	require.NoError(t, Register(Migration{Revision: 13}))
	require.NoError(t, Register(Migration{Revision: 14}))

	require.Len(t, migrations, 8)

	// Ensure migrations is maintained in sorted order
	prev := -1
	for _, m := range migrations {
		require.Greater(t, m.Revision, prev)
		prev = m.Revision
	}

	// Require an error when we register a duplicate migration
	require.Error(t, Register(Migration{Revision: 9}))
}

func TestRegisterDescriptor(t *testing.T) {
	defer Reset()

	require.NoError(t, RegisterDescriptor(generatedDescriptor))
	require.Len(t, migrations, 1)
}

var generatedDescriptor = []byte{
	// 405 bytes of compressed tidal.Descriptor data
	0x1f, 0x8b, 0x08, 0x08, 0x23, 0x3f, 0x35, 0x5f, 0x02, 0xff, 0x30, 0x30, 0x30, 0x31, 0x5f, 0x74,
	0x65, 0x73, 0x74, 0x5f, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x73, 0x71,
	0x6c, 0x00, 0x8c, 0x92, 0x41, 0x6f, 0x9c, 0x30, 0x10, 0x85, 0xef, 0xfc, 0x8a, 0xa7, 0x3d, 0x2d,
	0xd2, 0x72, 0x68, 0xa4, 0x48, 0xd5, 0xe6, 0x44, 0x59, 0x53, 0xa1, 0x52, 0x36, 0x65, 0x41, 0x6a,
	0x4e, 0x95, 0x81, 0x01, 0xac, 0x82, 0x8d, 0x6c, 0x93, 0xfe, 0xfd, 0x8a, 0x40, 0xb2, 0x6c, 0xd5,
	0xa6, 0x1c, 0xad, 0x79, 0xf3, 0xe6, 0xf3, 0x9b, 0xf1, 0x3c, 0x64, 0xad, 0x30, 0x10, 0x06, 0x1c,
	0x0d, 0x49, 0xd2, 0xbc, 0x43, 0xa9, 0xfa, 0x9e, 0xa4, 0x45, 0xad, 0x34, 0x6c, 0x4b, 0x20, 0x69,
	0x85, 0x26, 0xf4, 0xa2, 0xd1, 0xdc, 0x0a, 0x25, 0x51, 0x8b, 0x8e, 0x1c, 0xcf, 0xc3, 0xc0, 0xcb,
	0x9f, 0xbc, 0xa1, 0x23, 0x6a, 0xa5, 0xa6, 0xf7, 0xac, 0xa0, 0x23, 0xc6, 0x61, 0x7a, 0x26, 0xe7,
	0x8c, 0x1d, 0x61, 0xdf, 0xfc, 0xff, 0xf4, 0x1d, 0x87, 0xab, 0xe7, 0xa4, 0x0f, 0xe6, 0xba, 0x81,
	0x69, 0xd5, 0xd8, 0x55, 0x28, 0x08, 0x42, 0x96, 0xdd, 0x58, 0x51, 0x85, 0x62, 0xb4, 0x10, 0x8d,
	0x54, 0x9a, 0x2a, 0xc7, 0x09, 0x52, 0xe6, 0x67, 0x0c, 0x99, 0xff, 0x29, 0x66, 0x88, 0xc2, 0x69,
	0x10, 0xd8, 0xf7, 0xe8, 0x92, 0x5d, 0x30, 0x1a, 0xd2, 0x06, 0x7b, 0x07, 0x00, 0x76, 0xa2, 0xda,
	0x41, 0x48, 0x4b, 0x0d, 0x69, 0x3c, 0xa6, 0xd1, 0x57, 0x3f, 0x7d, 0xc2, 0x17, 0xf6, 0x74, 0x98,
	0xab, 0x93, 0x54, 0xf2, 0x9e, 0x76, 0x78, 0xe6, 0xba, 0x6c, 0xb9, 0xde, 0x7f, 0xb8, 0xfb, 0xe8,
	0xbe, 0x98, 0x25, 0x79, 0x1c, 0x23, 0x4f, 0xa2, 0x6f, 0x39, 0x5b, 0xc4, 0xd4, 0x73, 0xd1, 0x6d,
	0x51, 0xae, 0xe6, 0x60, 0x3f, 0x11, 0xb8, 0x8e, 0xfb, 0xf0, 0x2e, 0x72, 0xa3, 0xd5, 0x38, 0x6c,
	0x64, 0xbe, 0xe5, 0xbd, 0xbb, 0xbf, 0xff, 0x17, 0x2f, 0x2f, 0xad, 0x78, 0xa6, 0x1d, 0x0a, 0xa5,
	0x3a, 0xe2, 0x12, 0x27, 0x16, 0xfa, 0x79, 0x9c, 0xa1, 0xe6, 0x9d, 0xa1, 0xc3, 0xff, 0x90, 0x7a,
	0xea, 0x0b, 0xd2, 0xa6, 0x15, 0xc3, 0xf6, 0x28, 0x7f, 0xac, 0x25, 0xaf, 0x50, 0x4b, 0xfd, 0xe5,
	0x8b, 0xef, 0x08, 0xc2, 0x73, 0xca, 0xa2, 0xcf, 0xc9, 0x12, 0xda, 0xab, 0x9b, 0x8b, 0x94, 0x85,
	0x2c, 0x65, 0x49, 0xc0, 0x96, 0xc5, 0xce, 0x81, 0xfe, 0xad, 0xe7, 0x6d, 0xc2, 0x4d, 0xd3, 0x1c,
	0xed, 0xba, 0x6b, 0x0e, 0xe9, 0x3a, 0xe3, 0xb0, 0x82, 0x9b, 0x37, 0xb5, 0xbe, 0xe3, 0x4a, 0xfd,
	0x92, 0x5b, 0x2e, 0x79, 0xd2, 0xad, 0x6e, 0xd9, 0x39, 0xa5, 0xe7, 0xc7, 0x6b, 0xb0, 0x37, 0xa7,
	0x19, 0xf8, 0x97, 0xc0, 0x3f, 0xb1, 0x87, 0xdf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x3d, 0x3c, 0x7c,
	0xe3, 0x7a, 0x03, 0x00, 0x00,
}
