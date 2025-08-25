//go:build linux

package output_storage

// FIONREAD ioctl request code for Linux
// Source: asm-generic/ioctls.h (via <linux/termios.h>)
const FIONREAD = 0x541B
