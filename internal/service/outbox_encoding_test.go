package service_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"goroutine/internal/domain"
	"goroutine/internal/service"
	"goroutine/internal/testutil"
)

func TestParseNotificationPayload_Message(t *testing.T) {
	t.Parallel()

	recipient := testutil.ValidUserID()

	tests := []struct {
		name        string
		recipientID domain.UserID
		eventType   string
		payload     string
		wantMessage string
		wantIssues  []string
	}{
		{
			name:        "board created",
			recipientID: recipient,
			eventType:   "board.created",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work"}`,
			wantMessage: `a@x.com created board "Work"`,
		},
		{
			name:        "board updated",
			recipientID: recipient,
			eventType:   "board.updated",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work"}`,
			wantMessage: `a@x.com updated board "Work"`,
		},
		{
			name:        "board deleted",
			recipientID: recipient,
			eventType:   "board.deleted",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work"}`,
			wantMessage: `a@x.com deleted board "Work"`,
		},
		{
			name:        "column created",
			recipientID: recipient,
			eventType:   "column.created",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","columnName":"Todo"}`,
			wantMessage: `a@x.com created column "Todo" on board "Work"`,
		},
		{
			name:        "column updated",
			recipientID: recipient,
			eventType:   "column.updated",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","columnName":"Todo"}`,
			wantMessage: `a@x.com updated column "Todo" on board "Work"`,
		},
		{
			name:        "column moved",
			recipientID: recipient,
			eventType:   "column.moved",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","columnName":"Todo","sourcePosition":1,"targetPosition":3}`,
			wantMessage: `a@x.com moved column "Todo" on board "Work" from position 1 to 3`,
		},
		{
			name:        "column deleted",
			recipientID: recipient,
			eventType:   "column.deleted",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","columnName":"Todo"}`,
			wantMessage: `a@x.com deleted column "Todo" on board "Work"`,
		},
		{
			name:        "task created",
			recipientID: recipient,
			eventType:   "task.created",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","columnName":"Todo","taskName":"Ship"}`,
			wantMessage: `a@x.com created task "Ship" in column "Todo" on board "Work"`,
		},
		{
			name:        "task updated",
			recipientID: recipient,
			eventType:   "task.updated",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","columnName":"Todo","taskName":"Ship"}`,
			wantMessage: `a@x.com updated task "Ship" in column "Todo" on board "Work"`,
		},
		{
			name:        "task moved",
			recipientID: recipient,
			eventType:   "task.moved",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","taskName":"Ship","sourceColumnName":"Todo","targetColumnName":"Done","sourcePosition":2,"targetPosition":1}`,
			wantMessage: `a@x.com moved task "Ship" on board "Work" from "Todo" (2) to "Done" (1)`,
		},
		{
			name:        "task deleted",
			recipientID: recipient,
			eventType:   "task.deleted",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","columnName":"Todo","taskName":"Ship"}`,
			wantMessage: `a@x.com deleted task "Ship" in column "Todo" on board "Work"`,
		},
		{
			name:        "unknown type",
			recipientID: recipient,
			eventType:   "board.exploded",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work"}`,
			wantIssues:  []string{domain.ErrInvalidNotificationType},
		},
		{
			name:        "malformed payload",
			recipientID: recipient,
			eventType:   "board.created",
			payload:     `{`,
			wantIssues:  []string{domain.ErrInvalidNotificationPayload},
		},
		{
			name:        "invalid email",
			recipientID: recipient,
			eventType:   "board.created",
			payload:     `{"callerEmail":"not-an-email","boardName":"Work"}`,
			wantIssues:  []string{"Invalid email"},
		},
		{
			name:        "empty board name",
			recipientID: recipient,
			eventType:   "board.created",
			payload:     `{"callerEmail":"a@x.com","boardName":""}`,
			wantIssues:  []string{"Name is too short"},
		},
		{
			name:        "invalid column position",
			recipientID: recipient,
			eventType:   "column.moved",
			payload:     `{"callerEmail":"a@x.com","boardName":"Work","columnName":"Todo","sourcePosition":0,"targetPosition":1}`,
			wantIssues:  []string{domain.ErrColumnPositionValue},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			notificationType, err := domain.NewNotificationType(tt.eventType)

			var gotIssues []string
			if err != nil {
				gotIssues = domain.ExtractValidationIssues(err)
			} else {
				_, gotIssues = service.ParseNotificationPayload(notificationType, []byte(tt.payload))
			}

			diff := cmp.Diff(tt.wantIssues, gotIssues)
			if diff != "" {
				t.Errorf("got issues mismatch (-want +got):\n%s", diff)
				return
			}

			if tt.wantIssues != nil {
				return
			}

			parsed, _ := service.ParseNotificationPayload(notificationType, []byte(tt.payload))
			got := domain.Notification{
				RecipientID: tt.recipientID,
				Type:        notificationType,
				Payload:     parsed,
			}

			gotMessage := service.FormatNotificationMessage(got)
			if gotMessage != tt.wantMessage {
				t.Errorf("FormatNotificationMessage() = %q, want %q", gotMessage, tt.wantMessage)
			}
		})
	}
}
