package msiresult

// Exit codes returned by msiexec.
//
// https://learn.microsoft.com/en-us/windows/win32/msi/error-codes
const (
	Success                       ExitCode = 0    // ERROR_SUCCESS
	InvalidData                   ExitCode = 13   // ERROR_INVALID_DATA
	InvalidParameter              ExitCode = 87   // ERROR_INVALID_PARAMETER
	CallNotImplemented            ExitCode = 120  // ERROR_CALL_NOT_IMPLEMENTED
	ApphelpBlock                  ExitCode = 1259 // ERROR_APPHELP_BLOCK
	InstallServiceFailure         ExitCode = 1601 // ERROR_INSTALL_SERVICE_FAILURE
	InstallUserexit               ExitCode = 1602 // ERROR_INSTALL_USEREXIT
	InstallFailure                ExitCode = 1603 // ERROR_INSTALL_FAILURE
	InstallSuspend                ExitCode = 1604 // ERROR_INSTALL_SUSPEND
	UnknownProduct                ExitCode = 1605 // ERROR_UNKNOWN_PRODUCT
	UnknownFeature                ExitCode = 1606 // ERROR_UNKNOWN_FEATURE
	UnknownComponent              ExitCode = 1607 // ERROR_UNKNOWN_COMPONENT
	UnknownProperty               ExitCode = 1608 // ERROR_UNKNOWN_PROPERTY
	InvalidHandleState            ExitCode = 1609 // ERROR_INVALID_HANDLE_STATE
	BadConfiguration              ExitCode = 1610 // ERROR_BAD_CONFIGURATION
	IndexAbsent                   ExitCode = 1611 // ERROR_INDEX_ABSENT
	InstallSourceAbsent           ExitCode = 1612 // ERROR_INSTALL_SOURCE_ABSENT
	InstallPackageVersion         ExitCode = 1613 // ERROR_INSTALL_PACKAGE_VERSION
	ProductUninstalled            ExitCode = 1614 // ERROR_PRODUCT_UNINSTALLED
	BadQuerySyntax                ExitCode = 1615 // ERROR_BAD_QUERY_SYNTAX
	InvalidField                  ExitCode = 1616 // ERROR_INVALID_FIELD
	InstallAlreadyRunning         ExitCode = 1618 // ERROR_INSTALL_ALREADY_RUNNING
	InstallPackageOpenFailed      ExitCode = 1619 // ERROR_INSTALL_PACKAGE_OPEN_FAILED
	InstallPackageInvalid         ExitCode = 1620 // ERROR_INSTALL_PACKAGE_INVALID
	InstallUiFailure              ExitCode = 1621 // ERROR_INSTALL_UI_FAILURE
	InstallLogFailure             ExitCode = 1622 // ERROR_INSTALL_LOG_FAILURE
	InstallLanguageUnsupported    ExitCode = 1623 // ERROR_INSTALL_LANGUAGE_UNSUPPORTED
	InstallTransformFailure       ExitCode = 1624 // ERROR_INSTALL_TRANSFORM_FAILURE
	InstallPackageRejected        ExitCode = 1625 // ERROR_INSTALL_PACKAGE_REJECTED
	FunctionNotCalled             ExitCode = 1626 // ERROR_FUNCTION_NOT_CALLED
	FunctionFailed                ExitCode = 1627 // ERROR_FUNCTION_FAILED
	InvalidTable                  ExitCode = 1628 // ERROR_INVALID_TABLE
	DatatypeMismatch              ExitCode = 1629 // ERROR_DATATYPE_MISMATCH
	UnsupportedType               ExitCode = 1630 // ERROR_UNSUPPORTED_TYPE
	CreateFailed                  ExitCode = 1631 // ERROR_CREATE_FAILED
	InstallTempUnwritable         ExitCode = 1632 // ERROR_INSTALL_TEMP_UNWRITABLE
	InstallPlatformUnsupported    ExitCode = 1633 // ERROR_INSTALL_PLATFORM_UNSUPPORTED
	InstallNotused                ExitCode = 1634 // ERROR_INSTALL_NOTUSED
	PatchPackageOpenFailed        ExitCode = 1635 // ERROR_PATCH_PACKAGE_OPEN_FAILED
	PatchPackageInvalid           ExitCode = 1636 // ERROR_PATCH_PACKAGE_INVALID
	PatchPackageUnsupported       ExitCode = 1637 // ERROR_PATCH_PACKAGE_UNSUPPORTED
	ProductVersion                ExitCode = 1638 // ERROR_PRODUCT_VERSION
	InvalidCommandLine            ExitCode = 1639 // ERROR_INVALID_COMMAND_LINE
	InstallRemoteDisallowed       ExitCode = 1640 // ERROR_INSTALL_REMOTE_DISALLOWED
	SuccessRebootInitiated        ExitCode = 1641 // ERROR_SUCCESS_REBOOT_INITIATED
	PatchTargetNotFound           ExitCode = 1642 // ERROR_PATCH_TARGET_NOT_FOUND
	PatchPackageRejected          ExitCode = 1643 // ERROR_PATCH_PACKAGE_REJECTED
	InstallTransformRejected      ExitCode = 1644 // ERROR_INSTALL_TRANSFORM_REJECTED
	InstallRemoteProhibited       ExitCode = 1645 // ERROR_INSTALL_REMOTE_PROHIBITED
	PatchRemovalUnsupported       ExitCode = 1646 // ERROR_PATCH_REMOVAL_UNSUPPORTED
	UnknownPatch                  ExitCode = 1647 // ERROR_UNKNOWN_PATCH
	PatchNoSequence               ExitCode = 1648 // ERROR_PATCH_NO_SEQUENCE
	PatchRemovalDisallowed        ExitCode = 1649 // ERROR_PATCH_REMOVAL_DISALLOWED
	InvalidPatchXml               ExitCode = 1650 // ERROR_INVALID_PATCH_XML
	PatchManagedAdvertisedProduct ExitCode = 1651 // ERROR_PATCH_MANAGED_ADVERTISED_PRODUCT
	InstallServiceSafeboot        ExitCode = 1652 // ERROR_INSTALL_SERVICE_SAFEBOOT
	RollbackDisabled              ExitCode = 1653 // ERROR_ROLLBACK_DISABLED
	InstallRejected               ExitCode = 1654 // ERROR_INSTALL_REJECTED
	SuccessRebootRequired         ExitCode = 3010 // ERROR_SUCCESS_REBOOT_REQUIRED
)
