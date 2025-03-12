package envelope

//
//
//const (
//	statePlainText = iota
//	stateStartEncodedWord
//	stateEncodedWord
//	stateEncoding
//	stateCharset
//	statePayload
//	statePayloadEnd
//)
//
//// MimeHeaderDecode converts 7 bit encoded mime header strings to UTF-8
//func MimeHeaderDecode(str string) string {
//	// optimized to only create an output buffer if there's need to
//	// the `out` buffer is only made if an encoded word was decoded without error
//	// `out` is made with the capacity of len(str)
//	// a simple state machine is used to detect the start & end of encoded word and plain-text
//	state := statePlainText
//	var (
//		out        []byte
//		wordStart  int  // start of an encoded word
//		wordLen    int  // end of an encoded
//		ptextStart = -1 // start of plan-text
//		ptextLen   int  // end of plain-text
//	)
//	for i := 0; i < len(str); i++ {
//		switch state {
//		case statePlainText:
//			if ptextStart == -1 {
//				ptextStart = i
//			}
//			if str[i] == '=' {
//				state = stateStartEncodedWord
//				wordStart = i
//				wordLen = 1
//			} else {
//				ptextLen++
//			}
//		case stateStartEncodedWord:
//			if str[i] == '?' {
//				wordLen++
//				state = stateCharset
//			} else {
//				wordLen = 0
//				state = statePlainText
//				ptextLen++
//			}
//		case stateCharset:
//			if str[i] == '?' {
//				wordLen++
//				state = stateEncoding
//			} else if str[i] >= 'a' && str[i] <= 'z' ||
//				str[i] >= 'A' && str[i] <= 'Z' ||
//				str[i] >= '0' && str[i] <= '9' || str[i] == '-' {
//				wordLen++
//			} else {
//				// error
//				state = statePlainText
//				ptextLen += wordLen
//				wordLen = 0
//			}
//		case stateEncoding:
//			if str[i] == '?' {
//				wordLen++
//				state = statePayload
//			} else if str[i] == 'Q' || str[i] == 'q' || str[i] == 'b' || str[i] == 'B' {
//				wordLen++
//			} else {
//				// abort
//				state = statePlainText
//				ptextLen += wordLen
//				wordLen = 0
//			}
//
//		case statePayload:
//			if str[i] == '?' {
//				wordLen++
//				state = statePayloadEnd
//			} else {
//				wordLen++
//			}
//
//		case statePayloadEnd:
//			if str[i] == '=' {
//				wordLen++
//				var err error
//				out, err = decodeWordAppend(ptextLen, out, str, ptextStart, wordStart, wordLen)
//				if err != nil && out == nil {
//					// special case: there was an error with decoding and `out` wasn't created
//					// we can assume the encoded word as plaintext
//					ptextLen += wordLen //+ 1 // add 1 for the space/tab
//					wordLen = 0
//					wordStart = 0
//					state = statePlainText
//					continue
//				}
//				if skip := hasEncodedWordAhead(str, i+1); skip != -1 {
//					i = skip
//				} else {
//					out = makeAppend(out, len(str), []byte{})
//				}
//				ptextStart = -1
//				ptextLen = 0
//				wordLen = 0
//				wordStart = 0
//				state = statePlainText
//			} else {
//				// abort
//				state = statePlainText
//				ptextLen += wordLen
//				wordLen = 0
//			}
//
//		}
//	}
//
//	if out != nil && ptextLen > 0 {
//		out = makeAppend(out, len(str), []byte(str[ptextStart:ptextStart+ptextLen]))
//		ptextLen = 0
//	}
//
//	if out == nil {
//		// best case: there was nothing to encode
//		return str
//	}
//	return string(out)
//}
//
//func decodeWordAppend(ptextLen int, out []byte, str string, ptextStart int, wordStart int, wordLen int) ([]byte, error) {
//	if ptextLen > 0 {
//		out = makeAppend(out, len(str), []byte(str[ptextStart:ptextStart+ptextLen]))
//	}
//	d, err := Dec.Decode(str[wordStart : wordLen+wordStart])
//	if err == nil {
//		out = makeAppend(out, len(str), []byte(d))
//	} else if out != nil {
//		out = makeAppend(out, len(str), []byte(str[wordStart:wordLen+wordStart]))
//	}
//	return out, err
//}
//
//func makeAppend(out []byte, size int, in []byte) []byte {
//	if out == nil {
//		out = make([]byte, 0, size)
//	}
//	out = append(out, in...)
//	return out
//}
//
//func hasEncodedWordAhead(str string, i int) int {
//	for ; i+2 < len(str); i++ {
//		if str[i] != ' ' && str[i] != '\t' {
//			return -1
//		}
//		if str[i+1] == '=' && str[i+2] == '?' {
//			return i
//		}
//	}
//	return -1
//}
//
