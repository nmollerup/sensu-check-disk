package main

import (
	"testing"
)

// shouldSkipEntry determines if an fstab entry should be skipped during mount checking
func shouldSkipEntry(entry FstabEntry) bool {
	// Skip swap, bind mounts, noauto, and special filesystems
	if entry.FSType == "swap" {
		return true
	}

	// Skip comments, null mounts, and special entries
	if entry.MountPoint == "null" || entry.MountPoint == "" || len(entry.MountPoint) > 0 && entry.MountPoint[0] == '#' {
		return true
	}

	return false
}

func TestShouldSkipEntry_NullMountpoint(t *testing.T) {
	entry := FstabEntry{
		Device:     "/dev/sda1",
		MountPoint: "null",
		FSType:     "ext4",
		Options:    "defaults",
	}

	if !shouldSkipEntry(entry) {
		t.Error("expected entry with 'null' mountpoint to be skipped")
	}
}

func TestShouldSkipEntry_ValidMountpoint(t *testing.T) {
	entry := FstabEntry{
		Device:     "/dev/sda1",
		MountPoint: "/",
		FSType:     "ext4",
		Options:    "defaults",
	}

	if shouldSkipEntry(entry) {
		t.Error("expected entry with valid mountpoint '/' to not be skipped")
	}
}

func TestShouldSkipEntry_EmptyMountpoint(t *testing.T) {
	entry := FstabEntry{
		Device:     "/dev/sda1",
		MountPoint: "",
		FSType:     "ext4",
		Options:    "defaults",
	}

	if !shouldSkipEntry(entry) {
		t.Error("expected entry with empty mountpoint to be skipped")
	}
}

func TestShouldSkipEntry_CommentMountpoint(t *testing.T) {
	entry := FstabEntry{
		Device:     "/dev/sda1",
		MountPoint: "#/mnt/data",
		FSType:     "ext4",
		Options:    "defaults",
	}

	if !shouldSkipEntry(entry) {
		t.Error("expected entry with mountpoint starting with '#' to be skipped")
	}
}
