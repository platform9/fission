package v1

const (
	EXECUTOR_INSTANCEID_LABEL string = "executorInstanceId"
	POOLMGR_INSTANCEID_LABEL  string = "poolmgrInstanceId"
)

const (
	ChecksumTypeSHA256 ChecksumType = "sha256"
)

const (
	// ArchiveTypeLiteral means the package contents are specified in the Literal field of
	// resource itself.
	ArchiveTypeLiteral ArchiveType = "literal"

	// ArchiveTypeUrl means the package contents are at the specified URL.
	ArchiveTypeUrl ArchiveType = "url"
)

const (
	BuildStatusPending   = "pending"
	BuildStatusRunning   = "running"
	BuildStatusSucceeded = "succeeded"
	BuildStatusFailed    = "failed"
	BuildStatusNone      = "none"
)

const (
	AllowedFunctionsPerContainerSingle   = "single"
	AllowedFunctionsPerContainerInfinite = "infinite"
)

const (
	ExecutorTypePoolmgr   = "poolmgr"
	ExecutorTypeNewdeploy = "newdeploy"
)

const (
	StrategyTypeExecution = "execution"
)

const (
	SharedVolumeUserfunc   = "userfunc"
	SharedVolumePackages   = "packages"
	SharedVolumeSecrets    = "secrets"
	SharedVolumeConfigmaps = "configmaps"
)

const (
	MessageQueueTypeNats = "nats-streaming"
	MessageQueueTypeASQ  = "azure-storage-queue"
)

const (
	// FunctionReferenceFunctionName means that the function
	// reference is simply by function name.
	FunctionReferenceTypeFunctionName = "name"

	// Other function reference types we'd like to support:
	//   Versioned function, latest version
	//   Versioned function. by semver "latest compatible"
	//   Set of function references (recursively), by percentage of traffic
)
