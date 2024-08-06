package teamprojectsupdater_test

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/navikt/nada-backend/pkg/syncers/teamprojectsupdater"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type naisConsoleServiceMock struct {
	Invocation int
	Err        error
}

func (n *naisConsoleServiceMock) UpdateAllTeamProjects(_ context.Context) error {
	return n.Err
}

// Borrowed from: https://stackoverflow.com/a/36226525
type Buffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *Buffer) Read(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Read(p)
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p)
}

func (b *Buffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.String()
}

func TestTeamProjectsUpdater_Run(t *testing.T) {
	testCases := []struct {
		name          string
		err           error
		expectAtLeast int
		contains      string
	}{
		{
			name:          "no error",
			expectAtLeast: 2,
			contains:      "team projects updated",
		},
		{
			name:          "error",
			err:           fmt.Errorf("bob didnt build"),
			expectAtLeast: 2,
			contains:      "bob didnt build",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &Buffer{}

			m := &naisConsoleServiceMock{Err: tc.err}

			teamProjectsUpdater := teamprojectsupdater.New(m, zerolog.New(buf))

			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
			defer cancel()

			go teamProjectsUpdater.Run(ctx, 0*time.Second, 1*time.Second)
			time.Sleep(2 * time.Second)

			assert.GreaterOrEqual(t, tc.expectAtLeast, m.Invocation)
			assert.Contains(t, buf.String(), tc.contains)
		})
	}
}
