package tasks

import (
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/dnephin/buildpipe/config"
	//"github.com/fsouza/go-dockerclient"
)

// CommandTask is a task which runs a command in a container to produce a
// file or set of files.
type CommandTask struct {
	baseTask
	config *config.CommandConfig
}

// NewCommandTask creates a new CommandTask object
func NewCommandTask(options taskOptions, conf *config.CommandConfig) *CommandTask {
	return &CommandTask{
		baseTask: baseTask{
			name:   options.name,
			client: options.client,
		},
		config: conf,
	}
}

func (t *CommandTask) String() string {
	return fmt.Sprintf("CommandTask(name=%s, config=%s)", t.name, t.config)
}

func (t *CommandTask) logger() *log.Entry {
	return log.WithFields(log.Fields{
		"task":     "Command",
		"name":     t.name,
		"use":      t.config.Use,
		"command":  t.config.Command,
		"artifact": t.config.Artifact,
	})
}

// Run creates the host path if it doesn't already exist
func (t *CommandTask) Run(ctx *ExecuteContext) error {
	t.logger().Info("run")
	stale, err := t.isStale(ctx)
	if !stale || err != nil {
		return err
	}

	ctx.setModified(t.name)
	t.logger().Info("created")
	return nil
}

func (t *CommandTask) isStale(ctx *ExecuteContext) (bool, error) {
	if ctx.isModified(t.config.Dependencies()...) {
		return true, nil
	}

	if t.config.Artifact == "" {
		return true, nil
	}

	info, err := os.Stat(t.config.Artifact)
	// File or directory doesn't exist
	if err != nil {
		return true, nil
	}

	volumeFilesLastModified, err := t.volumeFilesLastModified(ctx)
	if err != nil {
		return true, err
	}

	if info.ModTime().Before(volumeFilesLastModified) {
		return true, nil
	}

	image, err := ctx.tasks.images[t.config.Use].getImage(ctx)
	if err != nil {
		return true, err
	}
	return info.ModTime().Before(image.Created), nil
}

// TODO: support a .volumeignore file?
func (t *CommandTask) volumeFilesLastModified(ctx *ExecuteContext) (time.Time, error) {
	// TODO: move this iteration to a more appropriate place
	volumePaths := []string{}
	for _, volumeName := range t.config.Volumes {
		// TODO: where does this name get validated that it is a volume?
		volume, _ := ctx.tasks.volumes[volumeName]
		volumePaths = append(volumePaths, volume.config.Path)
	}
	return lastModified(volumePaths...)
}
