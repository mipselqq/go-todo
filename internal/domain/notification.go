package domain

import (
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

func (t NotificationType) Value() string {
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
