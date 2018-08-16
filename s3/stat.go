package s3

type fileStat struct {
	name    string
	size    int64
	modTime time.Time
}

func (fs *fileStat) Name() string       { return fs.name }
func (fs *fileStat) Size() int64        { return fs.size }
func (*fileStat) Mode() FileMode        { return 0400 }
func (fs *fileStat) ModTime() time.Time { return fs.modTime }
func (*fileStat) IsDir() bool           { return false }
func (*fileStat) Sys() interface{}      { return nil }
