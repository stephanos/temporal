package agentworkflow

import (
	"time"

	"go.temporal.io/server/tools/agentworkflow/internal/recipe"
)

type Config struct {
	Root    string
	Backend Backend
	Limits  Limits
	Observe func(Progress)
}

type Progress struct {
	RunID   RunID   `json:"run_id"`
	State   string  `json:"state"`
	Phase   string  `json:"phase"`
	Outcome Outcome `json:"outcome,omitempty"`
}

type Limits struct {
	InvocationTimeout time.Duration `json:"invocation_timeout"`
	CheckTimeout      time.Duration `json:"check_timeout"`
	MaxOutputBytes    int64         `json:"max_output_bytes"`
	MaxEvents         int           `json:"max_events"`
	MaxSourceBytes    int64         `json:"max_source_bytes"`
	MaxSourceFiles    int           `json:"max_source_files"`
	MaxReviewers      int           `json:"max_reviewers"`
}

func DefaultLimits() Limits {
	return Limits{
		InvocationTimeout: 30 * time.Minute,
		CheckTimeout:      10 * time.Minute,
		MaxOutputBytes:    16 << 20,
		MaxEvents:         50_000,
		MaxSourceBytes:    2 << 30,
		MaxSourceFiles:    250_000,
		MaxReviewers:      4,
	}
}

type Request struct {
	Task     Task     `json:"task"`
	Project  Project  `json:"project"`
	Policy   Policy   `json:"policy"`
	Workflow Workflow `json:"workflow"`
}

type Workflow struct {
	Stages []WorkflowStage `json:"stages"`
}

type Models map[string]string

type WorkflowStage struct {
	Kind           StageKind `json:"kind"`
	Enabled        bool      `json:"enabled"`
	Models         Models    `json:"models,omitempty"`
	Prompt         string    `json:"prompt,omitempty"`
	ReviewPrompt   string    `json:"review_prompt,omitempty"`
	RevisionPrompt string    `json:"revision_prompt,omitempty"`
	Mode           string    `json:"mode,omitempty"`
}

type StageKind string

const (
	StageDiscover  StageKind = StageKind(recipe.Discover)
	StagePlan      StageKind = StageKind(recipe.Plan)
	StageImplement StageKind = StageKind(recipe.Implement)
	StageCheck     StageKind = StageKind(recipe.Check)
	StageReview    StageKind = StageKind(recipe.Review)
	StageRepair    StageKind = StageKind(recipe.Repair)
	StageApply     StageKind = StageKind(recipe.Apply)
)

func DefaultWorkflow() Workflow {
	return workflowFromRecipe(recipe.Default())
}

func normalizeWorkflow(workflow Workflow) (Workflow, error) {
	stages := make([]recipe.Stage, len(workflow.Stages))
	for index, stage := range workflow.Stages {
		stages[index] = recipe.Stage{
			Kind: recipe.Kind(stage.Kind), Enabled: stage.Enabled, Models: recipeModels(stage.Models), Prompt: stage.Prompt,
			ReviewPrompt: stage.ReviewPrompt, RevisionPrompt: stage.RevisionPrompt, Mode: stage.Mode,
		}
	}
	resolved, err := recipe.Normalize(recipe.Workflow{Stages: stages})
	if err != nil {
		return Workflow{}, err
	}
	return workflowFromRecipe(resolved), nil
}

func workflowFromRecipe(workflow recipe.Workflow) Workflow {
	result := Workflow{Stages: make([]WorkflowStage, len(workflow.Stages))}
	for index, stage := range workflow.Stages {
		result.Stages[index] = WorkflowStage{
			Kind: StageKind(stage.Kind), Enabled: stage.Enabled, Models: workflowModels(stage.Models), Prompt: stage.Prompt,
			ReviewPrompt: stage.ReviewPrompt, RevisionPrompt: stage.RevisionPrompt, Mode: stage.Mode,
		}
	}
	return result
}

func recipeModels(models Models) recipe.Models {
	if models == nil {
		return nil
	}
	result := make(recipe.Models, len(models))
	for provider, model := range models {
		result[provider] = model
	}
	return result
}

func workflowModels(models recipe.Models) Models {
	if models == nil {
		return nil
	}
	result := make(Models, len(models))
	for provider, model := range models {
		result[provider] = model
	}
	return result
}

func workflowStage(workflow Workflow, kind StageKind) WorkflowStage {
	if len(workflow.Stages) == 0 {
		workflow = DefaultWorkflow()
	}
	for _, stage := range workflow.Stages {
		if stage.Kind == kind {
			return stage
		}
	}
	return WorkflowStage{Kind: kind}
}

type Task struct {
	Objective       string   `json:"objective"`
	SuccessCriteria []string `json:"success_criteria"`
	Constraints     []string `json:"constraints,omitempty"`
	NonGoals        []string `json:"non_goals,omitempty"`
}

type Project struct {
	Root           string            `json:"root"`
	Source         SourcePolicy      `json:"source"`
	Instructions   []string          `json:"instructions,omitempty"`
	Checks         []Check           `json:"checks"`
	Environment    EnvironmentPolicy `json:"environment"`
	ForbiddenPaths []string          `json:"forbidden_paths,omitempty"`
}

type SourcePolicy struct {
	Mode    SourceMode `json:"mode"`
	Exclude []string   `json:"exclude,omitempty"`
}

type SourceMode string

const SourceDirectoryCopy SourceMode = "directory-copy"

type Check struct {
	Name      string        `json:"name"`
	Command   []string      `json:"command"`
	Directory string        `json:"directory,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
	Required  bool          `json:"required"`
}

type EnvironmentPolicy struct {
	Allow []string `json:"allow,omitempty"`
}

type Policy struct {
	Assurance        Assurance `json:"assurance"`
	MaxRepairs       int       `json:"max_repairs"`
	Reviewers        []string  `json:"reviewers,omitempty"`
	BlockingSeverity Severity  `json:"blocking_severity,omitempty"`
}

type Assurance string

const (
	AssuranceFast     Assurance = "fast"
	AssuranceStandard Assurance = "standard"
	AssuranceHigh     Assurance = "high"
)
