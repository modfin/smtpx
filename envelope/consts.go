package envelope

// MimeMultipartMixed is used for messages with multiple parts of different types.
// It allows combining different content types in a single message, such as
// text and attachments.
const MimeMultipartMixed = "multipart/mixed"

// MimeMultipartAlternative is used for presenting the same content in different formats.
// The client can choose the most appropriate format to display (e.g., plain text or HTML).
const MimeMultipartAlternative = "multipart/alternative"

// MimeMultipartRelated is used for messages with inline content, like HTML with embedded images.
// It groups related parts that should be considered as a single unit.
const MimeMultipartRelated = "multipart/related"

// MimeMultipartSigned is used for digitally signed emails.
// It contains two parts: the original message content and the digital signature.
const MimeMultipartSigned = "multipart/signed"

// MimeMultipartEncrypted is used for PGP/MIME encrypted messages.
// It typically contains two parts: the version information and the encrypted data.
const MimeMultipartEncrypted = "multipart/encrypted"

// MimeTextPlain is used for plain text content without any formatting.
const MimeTextPlain = "text/plain"

// MimeTextHtml is used for HTML-formatted content, allowing rich text and formatting.
const MimeTextHtml = "text/html"

// MimeTextEnriched is an obsolete format for rich text, predating HTML in emails.
const MimeTextEnriched = "text/enriched"

// MimeTextCalendar is used for iCalendar data, allowing the inclusion of calendar events.
const MimeTextCalendar = "text/calendar"

// MimeMessageEmail (message/rfc822) is used to embed entire email messages as attachments.
// It includes all headers and the body of the embedded email.
const MimeMessageEmail = "message/rfc822"

// MimeMessageDeliveryStatus is used for Delivery Status Notifications (DSNs),
// providing information about the delivery status of an email.
const MimeMessageDeliveryStatus = "message/delivery-status"

// MimeMessageDispositionNotification is used for Message Disposition Notifications (MDNs),
// indicating the recipient's handling of the message (e.g., displayed, deleted).
const MimeMessageDispositionNotification = "message/disposition-notification"

// MimeImageJpeg is used for JPEG image attachments or inline images.
const MimeImageJpeg = "image/jpeg"

// MimeImagePng is used for PNG image attachments or inline images.
const MimeImagePng = "image/png"

// MimeImageGif is used for GIF image attachments or inline images.
const MimeImageGif = "image/gif"

// MimeApplicationPdf is used for PDF document attachments.
const MimeApplicationPdf = "application/pdf"

// MimeApplicationOctet is a generic type for binary data when a more specific type is unknown.
const MimeApplicationOctet = "application/octet-stream"

// MimeApplicationPkcs7Mime is used for S/MIME encrypted or signed messages.
const MimeApplicationPkcs7Mime = "application/pkcs7-mime"

// MimeApplicationPgpEncrypted is found within the multipart/encrypted part,
// specifying the encryption method for PGP/MIME encrypted messages.
const MimeApplicationPgpEncrypted = "application/pgp-encrypted"

// MimeApplicationPgpSignature is used for PGP/MIME digitally signed messages.
// This contains the digital signature.
const MimeApplicationPgpSignature = "application/pgp-signature"
