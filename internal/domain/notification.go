package domain

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
