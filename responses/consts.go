package responses

const (
	// ClassSuccess specifies that the DSN is reporting a positive delivery
	// action.  Detail sub-codes may provide notification of
	// transformations required for delivery.
	ClassSuccess = 2
	// ClassTransientFailure - a persistent transient failure is one in which the message as
	// sent is valid, but persistence of some temporary condition has
	// caused abandonment or delay of attempts to send the message.
	// If this code accompanies a delivery failure report, sending in
	// the future may be successful.
	ClassTransientFailure = 4
	// ClassPermanentFailure - a permanent failure is one which is not likely to be resolved
	// by resending the message in the current form.  Some change to
	// the message or the destination must be made for successful
	// delivery.
	ClassPermanentFailure = 5
)

// space char
const SP = " "

// DefaultMap contains defined default codes (RfC 3463)
const (
	OtherStatus                             = ".0.0"
	OtherAddressStatus                      = ".1.0"
	BadDestinationMailboxAddress            = ".1.1"
	BadDestinationSystemAddress             = ".1.2"
	BadDestinationMailboxAddressSyntax      = ".1.3"
	DestinationMailboxAddressAmbiguous      = ".1.4"
	DestinationMailboxAddressValid          = ".1.5"
	MailboxHasMoved                         = ".1.6"
	BadSendersMailboxAddressSyntax          = ".1.7"
	BadSendersSystemAddress                 = ".1.8"
	OtherOrUndefinedMailboxStatus           = ".2.0"
	MailboxDisabled                         = ".2.1"
	MailboxFull                             = ".2.2"
	MessageLengthExceedsAdministrativeLimit = ".2.3"
	MailingListExpansionProblem             = ".2.4"
	OtherOrUndefinedMailSystemStatus        = ".3.0"
	MailSystemFull                          = ".3.1"
	SystemNotAcceptingNetworkMessages       = ".3.2"
	SystemNotCapableOfSelectedFeatures      = ".3.3"
	MessageTooBigForSystem                  = ".3.4"
	OtherOrUndefinedNetworkOrRoutingStatus  = ".4.0"
	NoAnswerFromHost                        = ".4.1"
	BadConnection                           = ".4.2"
	RoutingServerFailure                    = ".4.3"
	UnableToRoute                           = ".4.4"
	NetworkCongestion                       = ".4.5"
	RoutingLoopDetected                     = ".4.6"
	DeliveryTimeExpired                     = ".4.7"
	OtherOrUndefinedProtocolStatus          = ".5.0"
	InvalidCommand                          = ".5.1"
	SyntaxError                             = ".5.2"
	TooManyRecipients                       = ".5.3"
	InvalidCommandArguments                 = ".5.4"
	WrongProtocolVersion                    = ".5.5"
	OtherOrUndefinedMediaError              = ".6.0"
	MediaNotSupported                       = ".6.1"
	ConversionRequiredAndProhibited         = ".6.2"
	ConversionRequiredButNotSupported       = ".6.3"
	ConversionWithLossPerformed             = ".6.4"
	ConversionFailed                        = ".6.5"
)

var defaultTexts = struct {
	m map[EnhancedStatusCode]string
}{m: map[EnhancedStatusCode]string{
	EnhancedStatusCode{ClassSuccess, ".0.0"}:          "OK",
	EnhancedStatusCode{ClassSuccess, ".1.0"}:          "OK",
	EnhancedStatusCode{ClassSuccess, ".1.5"}:          "OK",
	EnhancedStatusCode{ClassSuccess, ".5.0"}:          "OK",
	EnhancedStatusCode{ClassTransientFailure, ".5.3"}: "Too many recipients",
	EnhancedStatusCode{ClassTransientFailure, ".5.4"}: "Relay access denied",
	EnhancedStatusCode{ClassPermanentFailure, ".5.1"}: "Invalid command",
}}

var codeMap = struct {
	m map[EnhancedStatusCode]int
}{m: map[EnhancedStatusCode]int{

	EnhancedStatusCode{ClassSuccess, OtherAddressStatus}:               250,
	EnhancedStatusCode{ClassSuccess, DestinationMailboxAddressValid}:   250,
	EnhancedStatusCode{ClassSuccess, OtherOrUndefinedMailSystemStatus}: 250,
	EnhancedStatusCode{ClassSuccess, OtherOrUndefinedProtocolStatus}:   250,
	EnhancedStatusCode{ClassSuccess, ConversionWithLossPerformed}:      250,
	EnhancedStatusCode{ClassSuccess, ".6.8"}:                           252,
	EnhancedStatusCode{ClassSuccess, ".7.0"}:                           220,

	EnhancedStatusCode{ClassTransientFailure, BadDestinationMailboxAddress}:      451,
	EnhancedStatusCode{ClassTransientFailure, BadSendersSystemAddress}:           451,
	EnhancedStatusCode{ClassTransientFailure, MailingListExpansionProblem}:       450,
	EnhancedStatusCode{ClassTransientFailure, OtherOrUndefinedMailSystemStatus}:  421,
	EnhancedStatusCode{ClassTransientFailure, MailSystemFull}:                    452,
	EnhancedStatusCode{ClassTransientFailure, SystemNotAcceptingNetworkMessages}: 453,
	EnhancedStatusCode{ClassTransientFailure, NoAnswerFromHost}:                  451,
	EnhancedStatusCode{ClassTransientFailure, BadConnection}:                     421,
	EnhancedStatusCode{ClassTransientFailure, RoutingServerFailure}:              451,
	EnhancedStatusCode{ClassTransientFailure, NetworkCongestion}:                 451,
	EnhancedStatusCode{ClassTransientFailure, OtherOrUndefinedProtocolStatus}:    451,
	EnhancedStatusCode{ClassTransientFailure, InvalidCommand}:                    430,
	EnhancedStatusCode{ClassTransientFailure, TooManyRecipients}:                 452,
	EnhancedStatusCode{ClassTransientFailure, InvalidCommandArguments}:           451,
	EnhancedStatusCode{ClassTransientFailure, ".7.0"}:                            450,
	EnhancedStatusCode{ClassTransientFailure, ".7.1"}:                            451,
	EnhancedStatusCode{ClassTransientFailure, ".7.12"}:                           422,
	EnhancedStatusCode{ClassTransientFailure, ".7.15"}:                           450,
	EnhancedStatusCode{ClassTransientFailure, ".7.24"}:                           451,

	EnhancedStatusCode{ClassPermanentFailure, BadDestinationMailboxAddress}:            550,
	EnhancedStatusCode{ClassPermanentFailure, BadDestinationMailboxAddressSyntax}:      501,
	EnhancedStatusCode{ClassPermanentFailure, BadSendersSystemAddress}:                 501,
	EnhancedStatusCode{ClassPermanentFailure, ".1.10"}:                                 556,
	EnhancedStatusCode{ClassPermanentFailure, MailboxFull}:                             552,
	EnhancedStatusCode{ClassPermanentFailure, MessageLengthExceedsAdministrativeLimit}: 552,
	EnhancedStatusCode{ClassPermanentFailure, OtherOrUndefinedMailSystemStatus}:        550,
	EnhancedStatusCode{ClassPermanentFailure, MessageTooBigForSystem}:                  552,
	EnhancedStatusCode{ClassPermanentFailure, RoutingServerFailure}:                    550,
	EnhancedStatusCode{ClassPermanentFailure, OtherOrUndefinedProtocolStatus}:          501,
	EnhancedStatusCode{ClassPermanentFailure, InvalidCommand}:                          500,
	EnhancedStatusCode{ClassPermanentFailure, SyntaxError}:                             500,
	EnhancedStatusCode{ClassPermanentFailure, InvalidCommandArguments}:                 501,
	EnhancedStatusCode{ClassPermanentFailure, ".5.6"}:                                  500,
	EnhancedStatusCode{ClassPermanentFailure, ConversionRequiredButNotSupported}:       554,
	EnhancedStatusCode{ClassPermanentFailure, ".6.6"}:                                  554,
	EnhancedStatusCode{ClassPermanentFailure, ".6.7"}:                                  553,
	EnhancedStatusCode{ClassPermanentFailure, ".6.8"}:                                  550,
	EnhancedStatusCode{ClassPermanentFailure, ".6.9"}:                                  550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.0"}:                                  550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.1"}:                                  551,
	EnhancedStatusCode{ClassPermanentFailure, ".7.2"}:                                  550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.4"}:                                  504,
	EnhancedStatusCode{ClassPermanentFailure, ".7.8"}:                                  554,
	EnhancedStatusCode{ClassPermanentFailure, ".7.9"}:                                  534,
	EnhancedStatusCode{ClassPermanentFailure, ".7.10"}:                                 523,
	EnhancedStatusCode{ClassPermanentFailure, ".7.11"}:                                 524,
	EnhancedStatusCode{ClassPermanentFailure, ".7.13"}:                                 525,
	EnhancedStatusCode{ClassPermanentFailure, ".7.14"}:                                 535,
	EnhancedStatusCode{ClassPermanentFailure, ".7.15"}:                                 550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.16"}:                                 552,
	EnhancedStatusCode{ClassPermanentFailure, ".7.17"}:                                 500,
	EnhancedStatusCode{ClassPermanentFailure, ".7.18"}:                                 500,
	EnhancedStatusCode{ClassPermanentFailure, ".7.19"}:                                 500,
	EnhancedStatusCode{ClassPermanentFailure, ".7.20"}:                                 550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.21"}:                                 550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.22"}:                                 550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.23"}:                                 550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.24"}:                                 550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.25"}:                                 550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.26"}:                                 550,
	EnhancedStatusCode{ClassPermanentFailure, ".7.27"}:                                 550,
}}
