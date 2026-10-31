package queue

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GoogleTasksClient wraps the Cloud Tasks client for a single pull‑queue.
// It provides helpers to enqueue, lease and acknowledge tasks.
// All tasks are expected to be processed by the internal worker service.
type GoogleTasksClient struct {
	client    *cloudtasks.Client
	projectID string
	location  string
	queueID   string
	queuePath string // full resource name: projects/{project}/locations/{location}/queues/{queue}
}

// NewGoogleTasksClient creates a new GoogleTasksClient.
// The caller can provide additional client options (e.g., credentials) via opts.
func NewGoogleTasksClient(ctx context.Context, projectID, location, queueID string, opts ...option.ClientOption) (*GoogleTasksClient, error) {
	if projectID == "" || location == "" || queueID == "" {
		return nil, fmt.Errorf("projectID, location and queueID must be provided")
	}

	c, err := cloudtasks.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Tasks client: %w", err)
	}

	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, location, queueID)

	return &GoogleTasksClient{
		client:    c,
		projectID: projectID,
		location:  location,
		queueID:   queueID,
		queuePath: queuePath,
	}, nil
}

// Close releases any resources held by the underlying Cloud Tasks client.
func (g *GoogleTasksClient) Close() error {
	return g.client.Close()
}

// EnqueueTask creates a new pull‑queue task.
// payload is the raw bytes that will be delivered to the worker.
// schedule (optional) defines when the task becomes eligible for lease.
// It returns the fully‑qualified task name.
func (g *GoogleTasksClient) EnqueueTask(ctx context.Context, payload []byte, schedule time.Time) (string, error) {
	if len(payload) == 0 {
		return "", fmt.Errorf("payload cannot be empty")
	}

	// Cloud Tasks expects the payload as raw bytes for pull queues.
	// Encode to base64 to stay safe with any binary data.
	encoded := base64.StdEncoding.EncodeToString(payload)

	task := &taskspb.Task{
		Payload: &taskspb.Task_PullMessage{
			PullMessage: &taskspb.PullMessage{
				Payload: []byte(encoded),
			},
		},
	}

	if !schedule.IsZero() {
		task.ScheduleTime = timestamppb.New(schedule)
	}

	req := &taskspb.CreateTaskRequest{
		Parent: g.queuePath,
		Task:   task,
	}

	created, err := g.client.CreateTask(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}
	return created.Name, nil
}

// LeaseTasks pulls up to maxTasks from the queue, holding them for leaseDuration.
// It returns the slice of leased tasks. The caller should process each task
// and subsequently call AcknowledgeTask (or DeleteTask on failure).
func (g *GoogleTasksClient) LeaseTasks(ctx context.Context, maxTasks int, leaseDuration time.Duration) ([]*taskspb.Task, error) {
	if maxTasks <= 0 {
		return nil, fmt.Errorf("maxTasks must be > 0")
	}
	if leaseDuration <= 0 {
		return nil, fmt.Errorf("leaseDuration must be > 0")
	}

	req := &taskspb.LeaseTasksRequest{
		Parent:        g.queuePath,
		MaxTasks:      int32(maxTasks),
		LeaseDuration: durationpb.New(leaseDuration),
	}

	resp, err := g.client.LeaseTasks(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to lease tasks: %w", err)
	}
	return resp.Tasks, nil
}

// AcknowledgeTask confirms successful processing of a leased task.
// The task argument must be one previously obtained from LeaseTasks,
// as the ScheduleTime field is required for the acknowledgement request.
func (g *GoogleTasksClient) AcknowledgeTask(ctx context.Context, task *taskspb.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}
	if task.Name == "" {
		return fmt.Errorf("task name is empty")
	}
	if task.ScheduleTime == nil {
		return fmt.Errorf("task schedule time is missing; cannot acknowledge")
	}

	req := &taskspb.AcknowledgeTaskRequest{
		Name:         task.Name,
		ScheduleTime: task.ScheduleTime,
	}

	if err := g.client.AcknowledgeTask(ctx, req); err != nil {
		return fmt.Errorf("failed to acknowledge task %s: %w", task.Name, err)
	}
	return nil
}

// DeleteTask removes a task from the queue regardless of its state.
// This is useful for tasks that have failed processing and should not be retried.
func (g *GoogleTasksClient) DeleteTask(ctx context.Context, taskName string) error {
	if taskName == "" {
		return fmt.Errorf("taskName cannot be empty")
	}
	req := &taskspb.DeleteTaskRequest{
		Name: taskName,
	}
	if err := g.client.DeleteTask(ctx, req); err != nil {
		return fmt.Errorf("failed to delete task %s: %w", taskName, err)
	}
	return nil
}

// DecodePayload reverses the base64 encoding applied in EnqueueTask.
// It returns the original payload bytes.
func DecodePayload(encoded []byte) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("failed to decode task payload: %w", err)
	}
	return decoded, nil
}