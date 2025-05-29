package lbdeployevent

import "github.com/leafbridge/leafbridge/core/lbevent"

// Registrations is an ordered list of all event registrations for the
// deployment system.
//
// The registrations can be provided to an [lbevent.Registry] to facilitate
// unmarshaling and event ID assignments.
var Registrations = []lbevent.Registration{
	{Type: FlowStartedType, Unmarshaler: lbevent.UnmarshalRecord[FlowStarted]},
	{Type: FlowStoppedType, Unmarshaler: lbevent.UnmarshalRecord[FlowStopped]},
	{Type: FlowConditionType, Unmarshaler: lbevent.UnmarshalRecord[FlowCondition]},
	{Type: FlowLockNotAcquiredType, Unmarshaler: lbevent.UnmarshalRecord[FlowLockNotAcquired]},
	{Type: FlowAlreadyRunningType, Unmarshaler: lbevent.UnmarshalRecord[FlowAlreadyRunning]},
	{Type: ActionStartedType, Unmarshaler: lbevent.UnmarshalRecord[ActionStarted]},
	{Type: ActionStoppedType, Unmarshaler: lbevent.UnmarshalRecord[ActionStopped]},
	{Type: CommandSkippedType, Unmarshaler: lbevent.UnmarshalRecord[CommandSkipped]},
	{Type: CommandStartedType, Unmarshaler: lbevent.UnmarshalRecord[CommandStarted]},
	{Type: CommandStoppedType, Unmarshaler: lbevent.UnmarshalRecord[CommandStopped]},
	{Type: DownloadStartedType, Unmarshaler: lbevent.UnmarshalRecord[DownloadStarted]},
	{Type: DownloadStoppedType, Unmarshaler: lbevent.UnmarshalRecord[DownloadStopped]},
	{Type: DownloadResetType, Unmarshaler: lbevent.UnmarshalRecord[DownloadReset]},
	{Type: ExtractionStartedType, Unmarshaler: lbevent.UnmarshalRecord[ExtractionStarted]},
	{Type: ExtractionStoppedType, Unmarshaler: lbevent.UnmarshalRecord[ExtractionStopped]},
	{Type: FileExtractionType, Unmarshaler: lbevent.UnmarshalRecord[FileExtraction]},
	{Type: FileVerificationType, Unmarshaler: lbevent.UnmarshalRecord[FileVerification]},
	{Type: FileCopyType, Unmarshaler: lbevent.UnmarshalRecord[FileCopy]},
	{Type: FileDeleteType, Unmarshaler: lbevent.UnmarshalRecord[FileDelete]},
}
