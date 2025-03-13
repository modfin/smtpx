package responses

// codeMap for mapping Enhanced Status Code to Basic Code
// Mapping according to https://www.iana.org/assignments/smtp-enhanced-status-codes/smtp-enhanced-status-codes.xml
// This might not be entirely useful

var FailLineTooLong = &response{
	enhancedCode: InvalidCommand,
	basicCode:    554,
	class:        ClassPermanentFailure,
	comment:      "Line too long.",
}

var FailNestedMailCmd = &response{
	enhancedCode: InvalidCommand,
	basicCode:    503,
	class:        ClassPermanentFailure,
	comment:      "Error: nested MAIL command",
}

var RejectedSenderMailCmd = &response{
	enhancedCode: InvalidCommandArguments,
	basicCode:    553,
	class:        ClassPermanentFailure,
	comment:      "Sender address rejected: Access denied",
}
var RejectedRcptCmd = &response{
	enhancedCode: InvalidCommandArguments,
	basicCode:    553,
	class:        ClassPermanentFailure,
	comment:      "Recipient address rejected: Access denied",
}

var SuccessMailCmd = &response{
	enhancedCode: OtherAddressStatus,
	class:        ClassSuccess,
}

var SuccessRcptCmd = &response{
	enhancedCode: DestinationMailboxAddressValid,
	class:        ClassSuccess,
}

var SuccessResetCmd = SuccessMailCmd

var SuccessNoopCmd = &response{
	enhancedCode: OtherStatus,
	class:        ClassSuccess,
}

var SuccessVerifyCmd = &response{
	enhancedCode: OtherOrUndefinedProtocolStatus,
	basicCode:    252,
	class:        ClassSuccess,
	comment:      "Cannot verify user",
}

var ErrorTooManyRecipients = &response{
	enhancedCode: TooManyRecipients,
	basicCode:    452,
	class:        ClassTransientFailure,
	comment:      "Too many recipients",
}

var ErrorRelayDenied = &response{
	enhancedCode: BadDestinationMailboxAddress,
	basicCode:    454,
	class:        ClassTransientFailure,
	comment:      "Error: Relay access denied:",
}

var SuccessQuitCmd = &response{
	enhancedCode: OtherStatus,
	basicCode:    221,
	class:        ClassSuccess,
	comment:      "Bye",
}

var FailNoSenderDataCmd = &response{
	enhancedCode: InvalidCommand,
	basicCode:    503,
	class:        ClassPermanentFailure,
	comment:      "Error: No sender",
}

var FailNoRecipientsDataCmd = &response{
	enhancedCode: InvalidCommand,
	basicCode:    503,
	class:        ClassPermanentFailure,
	comment:      "Error: No recipients",
}

var SuccessDataCmd = &response{
	basicCode: 354,
	comment:   "354 Enter message, ending with '.' on a line by itself",
}

var SuccessStartTLSCmd = &response{
	enhancedCode: OtherStatus,
	basicCode:    220,
	class:        ClassSuccess,
	comment:      "Ready to start TLS",
}

var FailUnrecognizedCmd = &response{
	enhancedCode: InvalidCommand,
	basicCode:    554,
	class:        ClassPermanentFailure,
	comment:      "Unrecognized command",
}

var FailMaxUnrecognizedCmd = &response{
	enhancedCode: InvalidCommand,
	basicCode:    554,
	class:        ClassPermanentFailure,
	comment:      "Too many unrecognized commands",
}

var ErrorShutdown = &response{
	enhancedCode: OtherOrUndefinedMailSystemStatus,
	basicCode:    421,
	class:        ClassTransientFailure,
	comment:      "Server is shutting down. Please try again later. Sayonara!",
}

var FailSyntaxError = &response{
	enhancedCode: SyntaxError,
	basicCode:    550,
	class:        ClassPermanentFailure,
	comment:      "Syntax error",
}

var FailReadLimitExceededDataCmd = &response{
	enhancedCode: MessageLengthExceedsAdministrativeLimit,
	basicCode:    550,
	class:        ClassPermanentFailure,
	comment:      "Error:",
}

var FailMessageSizeExceeded = &response{
	enhancedCode: OtherOrUndefinedNetworkOrRoutingStatus,
	basicCode:    552,
	class:        ClassPermanentFailure,
	comment:      "Error:",
}

var FailReadErrorDataCmd = &response{
	enhancedCode: OtherOrUndefinedMailSystemStatus,
	basicCode:    451,
	class:        ClassTransientFailure,
	comment:      "Error:",
}

var FailPathTooLong = &response{
	enhancedCode: InvalidCommandArguments,
	basicCode:    550,
	class:        ClassPermanentFailure,
	comment:      "Path too long",
}

var FailInvalidAddress = &response{
	enhancedCode: InvalidCommandArguments,
	basicCode:    501,
	class:        ClassPermanentFailure,
	comment:      "Invalid address",
}

var FailCommandNotImplemented = &response{
	enhancedCode: InvalidCommand,
	basicCode:    502,
	class:        ClassPermanentFailure,
	comment:      "Command not implemented",
}

var FailLocalPartTooLong = &response{
	enhancedCode: InvalidCommandArguments,
	basicCode:    550,
	class:        ClassPermanentFailure,
	comment:      "Local part too long, cannot exceed 64 characters",
}

var FailDomainTooLong = &response{
	enhancedCode: InvalidCommandArguments,
	basicCode:    550,
	class:        ClassPermanentFailure,
	comment:      "Domain cannot exceed 255 characters",
}

var FailBackendNotRunning = &response{
	enhancedCode: OtherOrUndefinedProtocolStatus,
	basicCode:    554,
	class:        ClassPermanentFailure,
	comment:      "Transaction failed - backend not running",
}

var FailBackendTransaction = &response{
	enhancedCode: OtherOrUndefinedProtocolStatus,
	basicCode:    554,
	class:        ClassPermanentFailure,
	comment:      "Error:",
}

var SuccessMessageQueued = &response{
	enhancedCode: OtherStatus,
	basicCode:    250,
	class:        ClassSuccess,
	comment:      "OK: queued",
}

var SuccessMessageAccepted = &response{
	enhancedCode: OtherStatus,
	basicCode:    250,
	class:        ClassSuccess,
	comment:      "Message accepted",
}

var FailBackendTimeout = &response{
	enhancedCode: OtherOrUndefinedProtocolStatus,
	basicCode:    554,
	class:        ClassPermanentFailure,
	comment:      "Error: transaction timeout",
}

var FailRcptCmd = &response{
	enhancedCode: BadDestinationMailboxAddress,
	basicCode:    550,
	class:        ClassPermanentFailure,
	comment:      "User unknown in local recipient table",
}

var FailMailCmd = &response{
	enhancedCode: BadDestinationMailboxAddress,
	basicCode:    550,
	class:        ClassPermanentFailure,
	comment:      "User unknown in local recipient table",
}
