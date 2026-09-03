package service

import (
	"encoding/json"

	"goroutine/internal/domain"
)

func ParseNotificationPayload(t domain.NotificationType, payloadBytes []byte) (payload domain.NotificationPayload, issues []string) {
	switch t.Value() {
	case domain.TypeBoardCreated, domain.TypeBoardUpdated, domain.TypeBoardDeleted:
		var wire struct {
			CallerEmail string `json:"callerEmail"`
			BoardName   string `json:"boardName"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{domain.ErrInvalidNotificationPayload}
		}

		callerEmail, err := domain.NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := domain.NewBoardName(wire.BoardName)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		switch t.Value() {
		case domain.TypeBoardCreated:
			return domain.BoardCreated{CallerEmail: callerEmail, BoardName: boardName}, nil
		case domain.TypeBoardUpdated:
			return domain.BoardUpdated{CallerEmail: callerEmail, BoardName: boardName}, nil
		default:
			return domain.BoardDeleted{CallerEmail: callerEmail, BoardName: boardName}, nil
		}

	case domain.TypeColumnCreated, domain.TypeColumnUpdated, domain.TypeColumnDeleted:
		var wire struct {
			CallerEmail string `json:"callerEmail"`
			BoardName   string `json:"boardName"`
			ColumnName  string `json:"columnName"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{domain.ErrInvalidNotificationPayload}
		}

		callerEmail, err := domain.NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := domain.NewBoardName(wire.BoardName)
		appendIssues(&issues, err)
		columnName, err := domain.NewColumnName(wire.ColumnName)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		switch t.Value() {
		case domain.TypeColumnCreated:
			return domain.ColumnCreated{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName}, nil
		case domain.TypeColumnUpdated:
			return domain.ColumnUpdated{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName}, nil
		default:
			return domain.ColumnDeleted{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName}, nil
		}

	case domain.TypeColumnMoved:
		var wire struct {
			CallerEmail    string `json:"callerEmail"`
			BoardName      string `json:"boardName"`
			ColumnName     string `json:"columnName"`
			SourcePosition int64  `json:"sourcePosition"`
			TargetPosition int64  `json:"targetPosition"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{domain.ErrInvalidNotificationPayload}
		}

		callerEmail, err := domain.NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := domain.NewBoardName(wire.BoardName)
		appendIssues(&issues, err)
		columnName, err := domain.NewColumnName(wire.ColumnName)
		appendIssues(&issues, err)
		sourcePosition, err := domain.NewColumnPosition(wire.SourcePosition)
		appendIssues(&issues, err)
		targetPosition, err := domain.NewColumnPosition(wire.TargetPosition)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		return domain.ColumnMoved{
			CallerEmail:    callerEmail,
			BoardName:      boardName,
			ColumnName:     columnName,
			SourcePosition: sourcePosition,
			TargetPosition: targetPosition,
		}, nil

	case domain.TypeTaskCreated, domain.TypeTaskUpdated, domain.TypeTaskDeleted:
		var wire struct {
			CallerEmail string `json:"callerEmail"`
			BoardName   string `json:"boardName"`
			ColumnName  string `json:"columnName"`
			TaskName    string `json:"taskName"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{domain.ErrInvalidNotificationPayload}
		}
		callerEmail, err := domain.NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := domain.NewBoardName(wire.BoardName)
		appendIssues(&issues, err)
		columnName, err := domain.NewColumnName(wire.ColumnName)
		appendIssues(&issues, err)
		taskName, err := domain.NewTaskName(wire.TaskName)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		switch t.Value() {
		case domain.TypeTaskCreated:
			return domain.TaskCreated{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName, TaskName: taskName}, nil
		case domain.TypeTaskUpdated:
			return domain.TaskUpdated{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName, TaskName: taskName}, nil
		default:
			return domain.TaskDeleted{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName, TaskName: taskName}, nil
		}

	case domain.TypeTaskMoved:
		var wire struct {
			CallerEmail      string `json:"callerEmail"`
			BoardName        string `json:"boardName"`
			TaskName         string `json:"taskName"`
			SourceColumnName string `json:"sourceColumnName"`
			TargetColumnName string `json:"targetColumnName"`
			SourcePosition   int64  `json:"sourcePosition"`
			TargetPosition   int64  `json:"targetPosition"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{domain.ErrInvalidNotificationPayload}
		}

		callerEmail, err := domain.NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := domain.NewBoardName(wire.BoardName)
		appendIssues(&issues, err)
		taskName, err := domain.NewTaskName(wire.TaskName)
		appendIssues(&issues, err)
		sourceColumnName, err := domain.NewColumnName(wire.SourceColumnName)
		appendIssues(&issues, err)
		targetColumnName, err := domain.NewColumnName(wire.TargetColumnName)
		appendIssues(&issues, err)
		sourcePosition, err := domain.NewTaskPosition(wire.SourcePosition)
		appendIssues(&issues, err)
		targetPosition, err := domain.NewTaskPosition(wire.TargetPosition)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		return domain.TaskMoved{
			CallerEmail:      callerEmail,
			BoardName:        boardName,
			TaskName:         taskName,
			SourceColumnName: sourceColumnName,
			TargetColumnName: targetColumnName,
			SourcePosition:   sourcePosition,
			TargetPosition:   targetPosition,
		}, nil

	default:
		return nil, []string{domain.ErrInvalidNotificationType}
	}
}

func appendIssues(issues *[]string, err error) {
	if err != nil {
		*issues = append(*issues, domain.ExtractValidationIssues(err)...)
	}
}
