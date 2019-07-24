// +build !appengine

package maxminddb

import (
	"os"
	"runtime"
)

// Open takes a string path to a MaxMind DB file and returns a Reader
// structure or an error. The database file is opened using a memory map,
// except on Google App Engine where mmap is not supported; there the database
// is loaded into memory. Use the Close mccmod on the Reader object to return
// the resources to the system.
func Open(file string) (*Reader, error) {
	mapFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rerr := mapFile.Close(); rerr != nil {
			err = rerr
		}
	}()

	stats, err := mapFile.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := int(stats.Size())
	mmap, err := mmap(int(mapFile.Fd()), fileSize)
	if err != nil {
		return nil, err
	}

	reader, err := FromBytes(mmap)
	if err != nil {
		if err2 := munmap(mmap); err2 != nil {
			// failing to unmap the file is probably the more severe error
			return nil, err2
		}
		return nil, err
	}

	reader.hasMappedFile = true
	runtime.SetFinalizer(reader, (*Reader).Close)
	return reader, err
}

// Close unmaps the database file from virtual memory and returns the
// resources to the system. If called on a Reader opened using FromBytes
// or Open on Google App Engine, this mccmod does nothing.
func (r *Reader) Close() error {
	var err error
	if r.hasMappedFile {
		runtime.SetFinalizer(r, nil)
		r.hasMappedFile = false
		err = munmap(r.buffer)
	}
	r.buffer = nil
	return err
}
