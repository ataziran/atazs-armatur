//go:build !linux && !darwin && !windows

package main

func platformCols() int { return 0 }
