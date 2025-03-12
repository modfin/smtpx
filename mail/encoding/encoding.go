// encoding enables using golang.org/x/net/html/charset for converting 7bit to UTF-8.
// golang.org/x/net/html/charset supports a larger range of encodings.
// when importing, place an underscore _ in front to import for side-effects

package encoding

import (
	"github.com/crholm/brevx/envelope"
	"io"

	cs "golang.org/x/net/html/charset"
)

func init() {

	envelope.Dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		return cs.NewReaderLabel(charset, input)
	}

}
