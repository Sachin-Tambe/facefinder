package model

import "time"

type User struct { ID string `json:"id,omitempty"`; Email string `json:"email,omitempty"`; Name string `json:"name,omitempty"` }
type Org struct { ID, Name, Slug, Plan, Role string; CreatedAt time.Time }
type Event struct {
	ID, OrganizationID, Name, Description, EventCode, RekognitionCollectionID, Status string
	StartsAt, EndsAt, ExpiresAt, FaceDataExpiresAt, PhotosExpireAt *time.Time
	GalleryTTLHours, WhatsAppSessionTTLHours, SelfieRetentionMinutes int
	AutoDeleteEnabled, DownloadsEnabled bool
	CreatedAt, UpdatedAt time.Time
	// populated only on create/detail responses for the owner
	EventToken, WhatsAppLink string `json:",omitempty"`
}
type Photo struct {
	ID, EventID, OrgID, OriginalKey string; WebKey, ThumbKey, AnalysisKey *string
	FileName, Mime string; FileSize int64; Width, Height int
	Status string; FaceCount int; Error *string
	UploadedAt time.Time; ProcessedAt *time.Time
}
type Face struct { ID, EventID, PhotoID, AWSFaceID, AWSImageID, BBoxJSON string; Confidence float64 }
type Search struct { ID, EventID, Phone, Status string; FaceCount *int; CreatedAt time.Time; CompletedAt *time.Time }
type SearchResult struct { PhotoID, BestFaceID string; Similarity float64; Type string; Rank int }
type Stats struct { TotalPhotos, Ready, Failed, NoFace, Processing, IndexedFaces, Searches, SuccessfulSearches, ZeroSearches int }
