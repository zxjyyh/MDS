package models

type UploadedFile struct {
	ID           int    `db:"id" json:"id"`
	OriginalName string `db:"original_name" json:"originalName"`
	StoredName   string `db:"stored_name" json:"storedName"`
	MimeType     string `db:"mime_type" json:"mimeType"`
	Size         int64  `db:"size" json:"size"`
	// StoragePath 是服务器磁盘上的绝对/相对路径，属于内部实现细节。
	// 一旦序列化给前端，等于把宿主机目录结构公开给所有登录用户，
	// 因此这里显式屏蔽（很多接口用 SELECT * + 直接返回 model）。
	StoragePath  string  `db:"storage_path" json:"-"`
	OwnerType    *string `db:"owner_type" json:"ownerType"`
	OwnerID      *int    `db:"owner_id" json:"ownerId"`
	UploadedByID int     `db:"uploaded_by" json:"uploadedById"`
	CreatedAt    string  `db:"created_at" json:"createdAt"`
}
