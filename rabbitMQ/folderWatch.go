package rabbitMQ

// FolderWatch -
type FolderWatch struct {
	Action string `json:"action"`
	Path   string `json:"path"`
	IsDir  string `json:"isDir"`
}

// CreateAction - represents a folder creation action
const CreateAction = "CREATE"

// DeleteAction - represents a folder deletion action
const DeleteAction = "DELETE"

// RenameAction - represents a folder renaming action
const RenameAction = "RENAME"

// MoveAction - represents a folder move action
const MoveAction = "MOVE"
