package edtech

import (
	"errors"
	"fmt"
	"time"
)

type Failure struct {
	CourseID      string    `json:"course_id"`
	LearnerID     string    `json:"learner_id"`
	DeliveryStage string    `json:"delivery_stage"`
	Deadline      time.Time `json:"deadline"`
	Message       string    `json:"message"`
	Exception     string    `json:"exception"`
}

type Capture struct {
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	Level       string         `json:"level"`
	Fingerprint []string       `json:"fingerprint"`
	Exception   string         `json:"exception"`
	Context     map[string]any `json:"context"`
}

type Decision struct {
	CourseID string `json:"course_id"`
	Action   string `json:"action"`
	Captured bool   `json:"captured"`
}

func Classify(now time.Time, f Failure) (Capture, Decision, error) {
	if f.CourseID == "" || f.LearnerID == "" || f.DeliveryStage == "" || f.Deadline.IsZero() || f.Message == "" || f.Exception == "" {
		return Capture{}, Decision{}, errors.New("course_id, learner_id, delivery_stage, deadline, message, and exception are required")
	}

	action := "delivery_retry"
	level := "error"
	if !f.Deadline.After(now.Add(24 * time.Hour)) {
		action = "educator_review"
		level = "warning"
	}

	capture := Capture{
		Title:       fmt.Sprintf("course %s delivery failed", f.CourseID),
		Message:     f.Message,
		Level:       level,
		Fingerprint: []string{"course-delivery", f.CourseID, f.DeliveryStage},
		Exception:   f.Exception,
		Context: map[string]any{
			"course_id":        f.CourseID,
			"learner_id":       f.LearnerID,
			"delivery_stage":   f.DeliveryStage,
			"deadline":         f.Deadline.UTC().Format(time.RFC3339),
			"reporting_action": action,
		},
	}
	return capture, Decision{CourseID: f.CourseID, Action: action, Captured: true}, nil
}
