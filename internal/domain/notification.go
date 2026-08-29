package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	TypeBoardCreated  = "board.created"
	TypeBoardUpdated  = "board.updated"
	TypeBoardDeleted  = "board.deleted"
	TypeColumnCreated = "column.created"
	TypeColumnUpdated = "column.updated"
	TypeColumnMoved   = "column.moved"
	TypeColumnDeleted = "column.deleted"
	TypeTaskCreated   = "task.created"
	TypeTaskUpdated   = "task.updated"
	TypeTaskMoved     = "task.moved"
	TypeTaskDeleted   = "task.deleted"

	ErrInvalidNotificationType    = "Invalid notification type"
	ErrInvalidNotificationPayload = "Invalid notification payload"

	ErrUnexpectedNotificationPayload = "unexpected notification payload"
)

type NotificationType struct {
	value string
}

func NewNotificationType(s string) (NotificationType, error) {
	switch s {
	case TypeBoardCreated, TypeBoardUpdated, TypeBoardDeleted,
		TypeColumnCreated, TypeColumnUpdated, TypeColumnMoved, TypeColumnDeleted,
		TypeTaskCreated, TypeTaskUpdated, TypeTaskMoved, TypeTaskDeleted:
		return NotificationType{value: s}, nil
	default:
		return NotificationType{}, &errValidation{Issues: []string{ErrInvalidNotificationType}}
	}
}

func (t NotificationType) String() string {
	return t.value
}

type NotificationPayload interface {
	notificationPayload()
}

type Notification struct {
	RecipientID UserID
	Type        NotificationType
	Payload     NotificationPayload
}

func NewNotification(recipientID UserID, eventType string, payload []byte) (Notification, error) {
	notificationType, err := NewNotificationType(eventType)
	if err != nil {
		return Notification{}, err
	}

	parsed, payloadIssues := parseNotificationPayload(notificationType, payload)
	if len(payloadIssues) > 0 {
		return Notification{}, &errValidation{Issues: payloadIssues}
	}

	return Notification{
		RecipientID: recipientID,
		Type:        notificationType,
		Payload:     parsed,
	}, nil
}

type BoardCreated struct {
	CallerEmail Email
	BoardName   BoardName
}

type BoardUpdated struct {
	CallerEmail Email
	BoardName   BoardName
}

type BoardDeleted struct {
	CallerEmail Email
	BoardName   BoardName
}

type ColumnCreated struct {
	CallerEmail Email
	BoardName   BoardName
	ColumnName  ColumnName
}

type ColumnUpdated struct {
	CallerEmail Email
	BoardName   BoardName
	ColumnName  ColumnName
}

type ColumnMoved struct {
	CallerEmail    Email
	BoardName      BoardName
	ColumnName     ColumnName
	SourcePosition ColumnPosition
	TargetPosition ColumnPosition
}

type ColumnDeleted struct {
	CallerEmail Email
	BoardName   BoardName
	ColumnName  ColumnName
}

type TaskCreated struct {
	CallerEmail Email
	BoardName   BoardName
	ColumnName  ColumnName
	TaskName    TaskName
}

type TaskUpdated struct {
	CallerEmail Email
	BoardName   BoardName
	ColumnName  ColumnName
	TaskName    TaskName
}

type TaskMoved struct {
	CallerEmail      Email
	BoardName        BoardName
	TaskName         TaskName
	SourceColumnName ColumnName
	TargetColumnName ColumnName
	SourcePosition   TaskPosition
	TargetPosition   TaskPosition
}

type TaskDeleted struct {
	CallerEmail Email
	BoardName   BoardName
	ColumnName  ColumnName
	TaskName    TaskName
}

func parseNotificationPayload(t NotificationType, payloadBytes []byte) (payload NotificationPayload, issues []string) {
	switch t.value {
	case TypeBoardCreated, TypeBoardUpdated, TypeBoardDeleted:
		var wire struct {
			CallerEmail string `json:"callerEmail"`
			BoardName   string `json:"boardName"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{ErrInvalidNotificationPayload}
		}

		callerEmail, err := NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := NewBoardName(wire.BoardName)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		switch t.value {
		case TypeBoardCreated:
			return BoardCreated{CallerEmail: callerEmail, BoardName: boardName}, nil
		case TypeBoardUpdated:
			return BoardUpdated{CallerEmail: callerEmail, BoardName: boardName}, nil
		default:
			return BoardDeleted{CallerEmail: callerEmail, BoardName: boardName}, nil
		}

	case TypeColumnCreated, TypeColumnUpdated, TypeColumnDeleted:
		var wire struct {
			CallerEmail string `json:"callerEmail"`
			BoardName   string `json:"boardName"`
			ColumnName  string `json:"columnName"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{ErrInvalidNotificationPayload}
		}

		callerEmail, err := NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := NewBoardName(wire.BoardName)
		appendIssues(&issues, err)
		columnName, err := NewColumnName(wire.ColumnName)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		switch t.value {
		case TypeColumnCreated:
			return ColumnCreated{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName}, nil
		case TypeColumnUpdated:
			return ColumnUpdated{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName}, nil
		default:
			return ColumnDeleted{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName}, nil
		}

	case TypeColumnMoved:
		var wire struct {
			CallerEmail    string `json:"callerEmail"`
			BoardName      string `json:"boardName"`
			ColumnName     string `json:"columnName"`
			SourcePosition int64  `json:"sourcePosition"`
			TargetPosition int64  `json:"targetPosition"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{ErrInvalidNotificationPayload}
		}

		callerEmail, err := NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := NewBoardName(wire.BoardName)
		appendIssues(&issues, err)
		columnName, err := NewColumnName(wire.ColumnName)
		appendIssues(&issues, err)
		sourcePosition, err := NewColumnPosition(wire.SourcePosition)
		appendIssues(&issues, err)
		targetPosition, err := NewColumnPosition(wire.TargetPosition)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		return ColumnMoved{
			CallerEmail:    callerEmail,
			BoardName:      boardName,
			ColumnName:     columnName,
			SourcePosition: sourcePosition,
			TargetPosition: targetPosition,
		}, nil

	case TypeTaskCreated, TypeTaskUpdated, TypeTaskDeleted:
		var wire struct {
			CallerEmail string `json:"callerEmail"`
			BoardName   string `json:"boardName"`
			ColumnName  string `json:"columnName"`
			TaskName    string `json:"taskName"`
		}
		err := json.Unmarshal(payloadBytes, &wire)
		if err != nil {
			return nil, []string{ErrInvalidNotificationPayload}
		}
		callerEmail, err := NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := NewBoardName(wire.BoardName)
		appendIssues(&issues, err)
		columnName, err := NewColumnName(wire.ColumnName)
		appendIssues(&issues, err)
		taskName, err := NewTaskName(wire.TaskName)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		switch t.value {
		case TypeTaskCreated:
			return TaskCreated{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName, TaskName: taskName}, nil
		case TypeTaskUpdated:
			return TaskUpdated{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName, TaskName: taskName}, nil
		default:
			return TaskDeleted{CallerEmail: callerEmail, BoardName: boardName, ColumnName: columnName, TaskName: taskName}, nil
		}

	case TypeTaskMoved:
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
			return nil, []string{ErrInvalidNotificationPayload}
		}

		callerEmail, err := NewEmail(wire.CallerEmail)
		appendIssues(&issues, err)
		boardName, err := NewBoardName(wire.BoardName)
		appendIssues(&issues, err)
		taskName, err := NewTaskName(wire.TaskName)
		appendIssues(&issues, err)
		sourceColumnName, err := NewColumnName(wire.SourceColumnName)
		appendIssues(&issues, err)
		targetColumnName, err := NewColumnName(wire.TargetColumnName)
		appendIssues(&issues, err)
		sourcePosition, err := NewTaskPosition(wire.SourcePosition)
		appendIssues(&issues, err)
		targetPosition, err := NewTaskPosition(wire.TargetPosition)
		appendIssues(&issues, err)

		if len(issues) > 0 {
			return nil, issues
		}

		return TaskMoved{
			CallerEmail:      callerEmail,
			BoardName:        boardName,
			TaskName:         taskName,
			SourceColumnName: sourceColumnName,
			TargetColumnName: targetColumnName,
			SourcePosition:   sourcePosition,
			TargetPosition:   targetPosition,
		}, nil

	default:
		return nil, []string{ErrInvalidNotificationType}
	}
}

func (n Notification) Message() (string, error) {
	switch p := n.Payload.(type) {
	case BoardCreated:
		return fmt.Sprintf("%s created board %q", p.CallerEmail, p.BoardName), nil
	case BoardUpdated:
		return fmt.Sprintf("%s updated board %q", p.CallerEmail, p.BoardName), nil
	case BoardDeleted:
		return fmt.Sprintf("%s deleted board %q", p.CallerEmail, p.BoardName), nil
	case ColumnCreated:
		return fmt.Sprintf("%s created column %q on board %q", p.CallerEmail, p.ColumnName, p.BoardName), nil
	case ColumnUpdated:
		return fmt.Sprintf("%s updated column %q on board %q", p.CallerEmail, p.ColumnName, p.BoardName), nil
	case ColumnMoved:
		return fmt.Sprintf("%s moved column %q on board %q from position %d to %d", p.CallerEmail, p.ColumnName, p.BoardName, p.SourcePosition.Int64(), p.TargetPosition.Int64()), nil
	case ColumnDeleted:
		return fmt.Sprintf("%s deleted column %q on board %q", p.CallerEmail, p.ColumnName, p.BoardName), nil
	case TaskCreated:
		return fmt.Sprintf("%s created task %q in column %q on board %q", p.CallerEmail, p.TaskName, p.ColumnName, p.BoardName), nil
	case TaskUpdated:
		return fmt.Sprintf("%s updated task %q in column %q on board %q", p.CallerEmail, p.TaskName, p.ColumnName, p.BoardName), nil
	case TaskMoved:
		return fmt.Sprintf("%s moved task %q on board %q from %q (%d) to %q (%d)", p.CallerEmail, p.TaskName, p.BoardName, p.SourceColumnName, p.SourcePosition.Int64(), p.TargetColumnName, p.TargetPosition.Int64()), nil
	case TaskDeleted:
		return fmt.Sprintf("%s deleted task %q in column %q on board %q", p.CallerEmail, p.TaskName, p.ColumnName, p.BoardName), nil
	default:
		return "", errors.New(ErrUnexpectedNotificationPayload)
	}
}

func (BoardCreated) notificationPayload()  {}
func (BoardUpdated) notificationPayload()  {}
func (BoardDeleted) notificationPayload()  {}
func (ColumnCreated) notificationPayload() {}
func (ColumnUpdated) notificationPayload() {}
func (ColumnMoved) notificationPayload()   {}
func (ColumnDeleted) notificationPayload() {}
func (TaskCreated) notificationPayload()   {}
func (TaskUpdated) notificationPayload()   {}
func (TaskMoved) notificationPayload()     {}
func (TaskDeleted) notificationPayload()   {}
