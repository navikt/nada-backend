package emulator

import (
	"net"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
)

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}

type Emulator struct {
	server *fakestorage.Server
	t      *testing.T
}

func (e *Emulator) ListObjectNames(bucket string) []string {
	objs, _, err := e.server.ListObjectsWithOptions(bucket, fakestorage.ListOptions{})
	if err != nil {
		e.t.Fatalf("getting objects in bucket %s: %v", bucket, err)
	}

	out := make([]string, len(objs))
	for i, obj := range objs {
		out[i] = obj.Name
	}

	return out
}

func (e *Emulator) GetObject(bucket, name string) fakestorage.Object {
	obj, err := e.server.GetObject(bucket, name)
	if err != nil {
		e.t.Fatalf("getting object %s/%s: %v", bucket, name, err)
	}

	return obj
}

func (e *Emulator) CreateBucket(name string) {
	e.server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{
		Name: name,
	})
}

func (e *Emulator) Client() *storage.Client {
	return e.server.Client()
}

func (e *Emulator) Cleanup() {
	e.server.Stop()
}

func New(t *testing.T, initialObjects []fakestorage.Object) *Emulator {
	t.Helper()

	port, err := GetFreePort()
	if err != nil {
		t.Fatalf("getting free port: %v", err)
	}

	server, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		InitialObjects: initialObjects,
		Scheme:         "http",
		Host:           "localhost",
		Port:           uint16(port),
	})
	if err != nil {
		t.Fatalf("creating fake storage server: %v", err)
	}

	return &Emulator{
		t:      t,
		server: server,
	}
}
