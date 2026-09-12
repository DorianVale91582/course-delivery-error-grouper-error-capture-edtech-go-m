package edtech

import (
	"reflect"
	"testing"
	"time"
)

func TestClassifyDeliveryFailure(t *testing.T) {
	now := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		deadline   time.Time
		wantAction string
		wantLevel  string
		wantGroup  []string
	}{
		{"deadline in six hours", now.Add(6 * time.Hour), "educator_review", "warning", []string{"course-delivery", "course-42", "lesson-publish"}},
		{"deadline in three days", now.Add(72 * time.Hour), "delivery_retry", "error", []string{"course-delivery", "course-42", "lesson-publish"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capture, decision, err := Classify(now, Failure{
				CourseID: "course-42", LearnerID: "learner-7", DeliveryStage: "lesson-publish",
				Deadline: tt.deadline, Message: "lesson package could not be delivered", Exception: "publish lesson: object validation failed",
			})
			if err != nil {
				t.Fatal(err)
			}
			if decision.Action != tt.wantAction || capture.Level != tt.wantLevel {
				t.Fatalf("action, level = %q, %q; want %q, %q", decision.Action, capture.Level, tt.wantAction, tt.wantLevel)
			}
			if !reflect.DeepEqual(capture.Fingerprint, tt.wantGroup) {
				t.Fatalf("fingerprint = %#v; want %#v", capture.Fingerprint, tt.wantGroup)
			}
		})
	}
}
