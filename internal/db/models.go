package db

import (
	"time"
)

// ProfessionalProject extends BaseProject from project-core with time tracking capabilities
type ProfessionalProject struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	BaseProjectID   string    `json:"baseProjectId" gorm:"uniqueIndex;not null"` // Links to project-core BaseProject
	Title           string    `json:"title"`                                     // Title
	ClientName      *string   `json:"clientName"`                                // Optional client (e.g., "THD" for TCS project)
	TotalSalaryCost float64   `json:"totalSalaryCost" gorm:"default:0"`          // Calculated field
	TotalHours      float64   `json:"totalHours" gorm:"default:0"`               // Calculated field
	IsActive        bool      `json:"isActive" gorm:"default:true"`              // Project status
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`

	// Relations
	ProjectAssignments []ProjectAssignment `json:"projectAssignments" gorm:"foreignKey:ParentProjectID"`
	TimeSessions       []TimeSession       `json:"timeSessions" gorm:"foreignKey:ProjectID"`
}

// ProjectAssignment represents one person's participation in a ProfessionalProject.
// Each assignment defines the hourly cost, activation status, and is linked to multiple work sessions.
// A single user may have multiple assignments to the same project if their rate or role changes over time.

type ProjectAssignment struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	ParentProjectID uint      `json:"parentProjectId" gorm:"not null"` // Links to ProfessionalProject
	WorkerUserID    string    `json:"workerUserId" gorm:"not null"`    // Single worker only (privacy model)
	CostPerHour     float64   `json:"costPerHour" gorm:"not null"`     // Freelance rate
	HoursDedicated  float64   `json:"hoursDedicated" gorm:"default:0"` // Calculated total
	TotalCost       float64   `json:"totalCost" gorm:"default:0"`      // Calculated: hours * rate
	Description     *string   `json:"description"`                     // Optional description
	IsActive        bool      `json:"isActive" gorm:"default:true"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`

	// Relations
	ParentProject ProfessionalProject `json:"parentProject" gorm:"foreignKey:ParentProjectID"`
	TimeSessions  []TimeSession       `json:"timeSessions" gorm:"foreignKey:ProjectAssignmentID"`
}

// TimeSession represents individual work sessions with detailed tracking
type TimeSession struct {
	ID                  uint       `json:"id" gorm:"primaryKey"`
	ProjectID           uint       `json:"projectId" gorm:"not null"` // Professional project ID
	ProjectAssignmentID *uint      `json:"projectAssignmentId"`       // Optional freelance sub-project
	UserID              string     `json:"userId" gorm:"not null"`    // Worker
	CompanyID           string     `json:"companyId" gorm:"not null"` // Company context
	StartTime           time.Time  `json:"startTime" gorm:"not null"`
	EndTime             *time.Time `json:"endTime"`                           // nil for active sessions
	SessionType         string     `json:"sessionType" gorm:"default:'work'"` // work, break, lunch, brb
	DurationMinutes     int        `json:"durationMinutes" gorm:"default:0"`  // Calculated duration
	HourlyRate          *float64   `json:"hourlyRate"`                        // Rate at time of session
	SessionCost         float64    `json:"sessionCost" gorm:"default:0"`      // Calculated cost
	Notes               *string    `json:"notes"`                             // Optional session notes
	IsActive            bool       `json:"isActive" gorm:"default:false"`     // Is currently active
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`

	// Relations
	Project           ProfessionalProject `json:"project" gorm:"foreignKey:ProjectID"`
	ProjectAssignment *ProjectAssignment  `json:"projectAssignment" gorm:"foreignKey:ProjectAssignmentID"`
}

// SessionBreak represents break periods within work sessions
type SessionBreak struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	SessionID       uint       `json:"sessionId" gorm:"not null"` // Parent session
	BreakType       string     `json:"breakType" gorm:"not null"` // break, lunch, brb
	StartTime       time.Time  `json:"startTime" gorm:"not null"`
	EndTime         *time.Time `json:"endTime"`                          // nil for active breaks
	DurationMinutes int        `json:"durationMinutes" gorm:"default:0"` // Calculated duration
	IsActive        bool       `json:"isActive" gorm:"default:false"`
	CreatedAt       time.Time  `json:"createdAt"`

	// Relations
	Session TimeSession `json:"session" gorm:"foreignKey:SessionID"`
}

// UserActiveSession tracks user's current active session (one per user max)
type UserActiveSession struct {
	UserID         string    `json:"userId" gorm:"primaryKey"`
	SessionID      uint      `json:"sessionId" gorm:"not null"`
	CompanyID      string    `json:"companyId" gorm:"not null"`
	ProjectID      uint      `json:"projectId" gorm:"not null"`
	StartedAt      time.Time `json:"startedAt" gorm:"not null"`
	LastActivityAt time.Time `json:"lastActivityAt" gorm:"not null"`
	IsOnBreak      bool      `json:"isOnBreak" gorm:"default:false"`
	CurrentBreakID *uint     `json:"currentBreakId"`
	UpdatedAt      time.Time `json:"updatedAt"`

	// Relations
	Session      TimeSession   `json:"session" gorm:"foreignKey:SessionID"`
	CurrentBreak *SessionBreak `json:"currentBreak" gorm:"foreignKey:CurrentBreakID"`
}

// ProjectTimeReport represents aggregated time data for reporting
type ProjectTimeReport struct {
	ProjectID      uint      `json:"projectId"`
	ProjectTitle   string    `json:"projectTitle"`
	CompanyID      string    `json:"companyId"`
	TotalHours     float64   `json:"totalHours"`
	TotalCost      float64   `json:"totalCost"`
	WorkSessions   int       `json:"workSessions"`
	AverageSession float64   `json:"averageSession"` // in hours
	LastActivity   time.Time `json:"lastActivity"`
	ActiveWorkers  int       `json:"activeWorkers"`
}

// UserTimeReport represents individual user time tracking data
type UserTimeReport struct {
	UserID          string    `json:"userId"`
	ProjectID       uint      `json:"projectId"`
	CompanyID       string    `json:"companyId"`
	TotalHours      float64   `json:"totalHours"`
	WorkSessions    int       `json:"workSessions"`
	BreakMinutes    int       `json:"breakMinutes"`
	ProductiveHours float64   `json:"productiveHours"` // total - breaks
	LastSession     time.Time `json:"lastSession"`
	AverageDaily    float64   `json:"averageDaily"` // hours per day
}

// SessionType constants
const (
	SessionTypeWork  = "work"
	SessionTypeBreak = "break"
	SessionTypeLunch = "lunch"
	SessionTypeBRB   = "brb"
)

// BreakType constants
const (
	BreakTypeShort = "break" // 5-15 minutes
	BreakTypeLunch = "lunch" // 30-60 minutes
	BreakTypeBRB   = "brb"   // 1-5 minutes (bathroom, quick interruption)
)
