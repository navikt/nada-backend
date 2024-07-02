package teamprojectsupdater_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/navikt/nada-backend/pkg/syncers/teamprojectsupdater"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type naisConsoleServiceMock struct {
	Invocation int
	Err        error
}

func (n *naisConsoleServiceMock) UpdateAllTeamProjects(_ context.Context) error {
	return n.Err
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
			buf := &bytes.Buffer{}

			m := &naisConsoleServiceMock{Err: tc.err}

			teamProjectsUpdater := teamprojectsupdater.New(m, zerolog.New(buf))

			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
			defer cancel()

			go teamProjectsUpdater.Run(ctx, 0*time.Second, 1*time.Second)
			time.Sleep(2 * time.Second)

			assert.GreaterOrEqual(t, tc.expectAtLeast, m.Invocation)
			assert.Contains(t, buf.String(), tc.contains)
			fmt.Println("logs: ", buf.String())
		})
	}
}
