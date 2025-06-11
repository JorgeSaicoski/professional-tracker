# Professional Tracker Module

Advanced time tracking and project cost management for professional work environments. Part of the Personal Manager microservices ecosystem.

## 🎯 Purpose

Professional Tracker handles complex time tracking scenarios where users work across multiple companies, projects, and roles simultaneously. It provides detailed work session management, cost calculations, and privacy-focused reporting.

## 🏗️ Architecture Role

This module is part of the **Personal Manager microservices architecture**:

```
Personal Manager System
├── project-core ✅         ← Base project functionality
├── professional-tracker ⏳  ← This module - Time tracking & costs
├── education-manager ⏳     ← Course & student management  
└── finance-tracker ⏳       ← Financial goals & investments
```

Professional Tracker **extends** Project-Core by adding time tracking capabilities to base projects.

## 🎪 Target Users

### Freelancers
- Track time across multiple clients
- Calculate project costs automatically
- Manage work sessions for different projects
- Generate client billing reports

### Remote Workers
- Track time for multiple employers simultaneously
- Manage work sessions across different companies
- Sequential work scheduling (Company A → Freelance → Studies)
- Break and interruption tracking

### Companies & Teams
- Monitor employee time allocation to projects
- Calculate real project costs based on tracked time
- Analyze team productivity and resource allocation
- Privacy-focused reporting (owners see costs, not individual details)

### Consultants & Contractors
- Bill clients based on accurately tracked time
- Manage multiple concurrent projects
- Track different types of work sessions
- Generate detailed time reports

## 🚀 Key Features

### Multi-Company Time Tracking
- **Simultaneous Company Work**: Users can work for multiple companies with one account
- **Sequential Sessions**: Switch between company work → freelance → personal projects
- **Company Isolation**: Each company sees only their own project data
- **Role Flexibility**: Same user can be employee at Company A, freelancer for Company B

### Advanced Work Session Management

**Session Types:**
- **Work Sessions**: Active project work time
- **Break Sessions**: Short breaks, lunch breaks, bathroom breaks (brb)
- **Project Switching**: Change projects within same company
- **Company Switching**: Move between different companies/employers

**Work Flow Example:**
```
9:00 AM  → Start work (TCS Company, Project Alpha)
10:30 AM → Take break (coffee break)
10:45 AM → Resume work (TCS Company, Project Alpha)
12:00 PM → Lunch break
1:00 PM  → Resume work (TCS Company, Project Beta) [project switch]
5:00 PM  → Finish company work
6:00 PM  → Start freelance work (Personal Client, Website Project)
8:00 PM  → Finish day
```

### Project Cost Management

**Company Projects:**
- Track employee time allocation to projects
- Calculate real project costs: `time_spent × employee_hourly_rate`
- Multiple employees can work on same project
- Client assignment (e.g., "TCS project for THD client")

**Freelance Sub-Projects:**
- One worker per freelance project (privacy model)
- Direct cost calculation: `hours_worked × freelance_rate`
- Independent billing and tracking

**Privacy Protection:**
- **Company View**: Total project costs, team productivity metrics
- **Individual Privacy**: Personal work details remain private
- **Cost-Only Reporting**: Companies see expenses, not individual time details

### Time Session Controls

**Active Session Management:**
- Start/stop work sessions
- Pause for breaks (with break type tracking)
- Switch between projects seamlessly
- Emergency session recovery (if app crashes during work)

**Session History:**
- Complete work history across all companies
- Break time analysis and patterns
- Project time distribution
- Daily/weekly/monthly summaries

## 📊 Data Models

### Core Models

```go
type ProfessionalProject struct {
    BaseProjectID   string    // Links to project-core BaseProject
    ClientName      *string   // Optional client (e.g., "THD" for TCS project)
    SalaryPerHour   *float64  // For cost calculations
    TotalSalaryCost float64   // Calculated: sum(time_sessions * hourly_rate)
    
    // Relations
    FreelanceProjects []FreelanceProject  // Sub-projects for freelance work
    TimeSessions     []TimeSession       // All work sessions
}

type FreelanceProject struct {
    ParentProjectID string  // Links to ProfessionalProject
    WorkerUserID    string  // Single worker only (privacy)
    CostPerHour     float64 // Freelance rate
    HoursDedicated  float64 // Calculated total
    TotalCost       float64 // Calculated: hours * rate
}

type TimeSession struct {
    ID          uint      // Primary key
    ProjectID   string    // Professional project ID
    UserID      string    // Worker
    CompanyID   string    // Company context
    StartTime   time.Time
    EndTime     *time.Time // nil for active sessions
    SessionType string    // "work", "break", "lunch", "brb"
    
    // Methods
    GetDuration() time.Duration
    IsActive() bool
}
```

### Session Types

- **`work`**: Active project work
- **`break`**: Short break (5-15 minutes)
- **`lunch`**: Lunch break (30-60 minutes)
- **`brb`**: Bathroom/quick interruption (1-5 minutes)

## 🔌 API Endpoints

### Session Management
```http
# Start work session
POST /api/internal/professional/sessions/start
{
  "projectId": "prof-123",
  "companyId": "company-456",
  "userId": "user-789"
}

# Take break
POST /api/internal/professional/sessions/break
{
  "sessionId": "session-123",
  "breakType": "lunch"
}

# Switch project (within same company)
POST /api/internal/professional/sessions/switch-project
{
  "currentSessionId": "session-123",
  "newProjectId": "prof-456"
}

# Switch company (finish current, start new)
POST /api/internal/professional/sessions/switch-company
{
  "currentSessionId": "session-123",
  "newCompanyId": "company-789",
  "newProjectId": "prof-999"
}

# Finish work day
POST /api/internal/professional/sessions/finish
{
  "sessionId": "session-123"
}
```

### Reporting & Analytics
```http
# Get user's active session
GET /api/internal/professional/sessions/active?userId=user-123

# Get session history
GET /api/internal/professional/sessions/history?userId=user-123&startDate=2025-01-01&endDate=2025-01-31

# Get project time summary
GET /api/internal/professional/projects/{projectId}/time-summary

# Get company cost report (owners only)
GET /api/internal/professional/companies/{companyId}/cost-report?startDate=2025-01-01
```

### Project Management
```http
# Create professional project
POST /api/internal/professional/projects
{
  "baseProjectId": "base-123",
  "clientName": "THD Corp",
  "salaryPerHour": 50.0
}

# Get project details
GET /api/internal/professional/projects/{projectId}

# Update project
PUT /api/internal/professional/projects/{projectId}

# Add freelance sub-project
POST /api/internal/professional/projects/{projectId}/freelance
{
  "workerUserId": "user-456",
  "costPerHour": 75.0
}
```

## 🔧 Integration with Project-Core

Professional Tracker **extends** base projects from Project-Core:

### Data Relationship
```go
// Project-Core provides:
type BaseProject struct {
    ID          uint
    Title       string
    Description *string
    OwnerID     string
    CompanyID   *string
    // ... base fields
}

// Professional Tracker extends:
type ProfessionalProject struct {
    BaseProjectID string  // References BaseProject.ID
    // ... professional-specific fields
}
```

### API Integration
```go
// Professional Tracker calls Project-Core APIs:
func (s *ProfessionalService) CreateProfessionalProject(req *CreateRequest) {
    // 1. Validate base project exists
    baseProject := s.projectCoreClient.GetProject(req.BaseProjectID)
    
    // 2. Check permissions
    canAccess := s.projectCoreClient.UserCanAccessProject(userID, baseProject.ID)
    
    // 3. Create professional extension
    profProject := &ProfessionalProject{
        BaseProjectID: req.BaseProjectID,
        // ... professional fields
    }
}
```

## 🔒 Security & Privacy

### Permission Model
- **Project Access**: Must have access to base project in Project-Core
- **Company Scope**: Users only see data for companies they belong to
- **Session Privacy**: Individual work sessions are private to the worker
- **Cost Transparency**: Companies see aggregated costs, not detailed time logs

### Data Privacy Rules
1. **Individual Sessions**: Only the worker can see their detailed time logs
2. **Company Reports**: Owners see total costs and hours, not individual breakdowns
3. **Client Information**: Only project members see client details
4. **Cross-Company Isolation**: Company A cannot see Company B data

### Authentication
- Uses same Keycloak integration as other services
- JWT token validation for all endpoints
- User context extraction from tokens
- Company membership validation via Project-Core

## 🚀 Getting Started

### Prerequisites
- Go 1.23+
- PostgreSQL database
- Project-Core service running
- Keycloak authentication service

### Installation
```bash
git clone https://github.com/JorgeSaicoski/professional-tracker.git
cd professional-tracker
go mod download
```

### Configuration
```bash
# Environment variables
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=yourpassword
export POSTGRES_DB=professional_tracker_db

# Keycloak configuration
export KEYCLOAK_URL=http://keycloak:8080/keycloak
export KEYCLOAK_REALM=master

# Project-Core integration
export PROJECT_CORE_URL=http://project-core:8001/api/internal
```

### Database Migration
```bash
# Models auto-migrate on startup:
# - ProfessionalProject
# - FreelanceProject  
# - TimeSession
go run cmd/server/main.go
```

### Running the Service
```bash
# Development
go run cmd/server/main.go

# Production
go build -o professional-tracker cmd/server/main.go
./professional-tracker
```

## 📱 Usage Examples

### Freelancer Workflow
```bash
# 1. Start freelance work
curl -X POST /api/internal/professional/sessions/start \
  -d '{"projectId":"freelance-web-design","companyId":"personal","userId":"freelancer-123"}'

# 2. Take lunch break
curl -X POST /api/internal/professional/sessions/break \
  -d '{"sessionId":"session-456","breakType":"lunch"}'

# 3. Resume work
curl -X POST /api/internal/professional/sessions/resume \
  -d '{"sessionId":"session-456"}'

# 4. Finish work
curl -X POST /api/internal/professional/sessions/finish \
  -d '{"sessionId":"session-456"}'
```

### Multi-Company Employee
```bash
# 1. Start morning work at Company A
curl -X POST /api/internal/professional/sessions/start \
  -d '{"projectId":"companyA-project","companyId":"company-a","userId":"employee-123"}'

# 2. Finish Company A work and switch to freelance
curl -X POST /api/internal/professional/sessions/switch-company \
  -d '{"currentSessionId":"session-789","newCompanyId":"personal","newProjectId":"freelance-project"}'

# 3. Later switch to evening studies
curl -X POST /api/internal/professional/sessions/switch-company \
  -d '{"currentSessionId":"session-890","newCompanyId":"university","newProjectId":"thesis-project"}'
```

## 📊 Real-World Use Cases

### Case 1: Software Developer
**Scenario**: Works full-time at TCS, freelances web development, studies computer science

**Daily Flow**:
- 9:00 AM - 5:00 PM: TCS Company (various projects)
- 6:00 PM - 8:00 PM: Freelance web development
- 8:30 PM - 10:00 PM: University coursework

**Benefits**:
- Accurate time tracking for each context
- Separate billing for freelance work
- Study time tracking for personal goals
- No cross-contamination of company data

### Case 2: Dance Instructor
**Scenario**: Teaches at dance school, runs private lessons, manages studio

**Projects**:
- **School Employee**: Teaching scheduled classes
- **Private Instructor**: Individual student sessions
- **Studio Owner**: Administrative and management tasks

**Benefits**:
- Track teaching hours vs administrative time
- Calculate income from different sources
- Manage multiple revenue streams
- Professional growth analytics

### Case 3: Consulting Firm
**Scenario**: Team of consultants working on multiple client projects

**Structure**:
- Multiple consultants per project
- Different hourly rates per consultant
- Client-specific project tracking
- Resource allocation optimization

**Benefits**:
- Real project costs (not estimates)
- Resource utilization reports
- Client billing accuracy
- Team productivity insights

## 🔄 Development Roadmap

### Phase 1: Core Time Tracking ⏳
- Basic session start/stop functionality
- Work/break session types
- Single project time tracking
- User session history

### Phase 2: Multi-Company Support ⏳
- Company context for sessions
- Project switching within companies
- Company switching workflow
- Cross-company data isolation

### Phase 3: Advanced Features ⏳
- Freelance sub-project management
- Cost calculation automation
- Break time analysis
- Session recovery mechanisms

### Phase 4: Reporting & Analytics ⏳
- Company cost reports
- Individual productivity analytics
- Time pattern analysis
- Export capabilities

### Phase 5: Integration & Polish ⏳
- Calendar service integration
- Mobile app support
- Real-time session sync
- Advanced reporting dashboard

## 🤝 Contributing

This module is part of the larger Personal Manager ecosystem. Development guidelines:

1. **Follow Project-Core patterns** for API design and data modeling
2. **Maintain privacy boundaries** between companies and users
3. **Use shared libraries** (pgconnect, microservice-commons, keycloak-auth)
4. **Test multi-company scenarios** thoroughly
5. **Document privacy implications** of new features

### Development Setup
```bash
# Clone with dependencies
git clone --recurse-submodules https://github.com/JorgeSaicoski/personal-manager.git

# Start supporting services
cd personal-manager
podman compose up project-core keycloak db

# Develop professional-tracker
cd professional-tracker
go run cmd/server/main.go
```

## 📞 Support & Documentation

- **Main Project**: [Personal Manager](https://github.com/JorgeSaicoski/personal-manager)
- **Project-Core**: [Base project functionality](https://github.com/JorgeSaicoski/go-project-manager)
- **Shared Libraries**: 
  - [pgconnect](https://github.com/JorgeSaicoski/pgconnect) - Database operations
  - [keycloak-auth](https://github.com/JorgeSaicoski/keycloak-auth) - Authentication
  - [microservice-commons](https://github.com/JorgeSaicoski/microservice-commons) - Common utilities

## 📄 License

MIT License - See [LICENSE](LICENSE) for details.

---

**Professional Tracker** - Where work meets precision. Track every moment, optimize every hour, achieve professional excellence.
