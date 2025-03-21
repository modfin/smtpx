package envelope

import (
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io"
	"strings"
)

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	charset = strings.ToLower(charset)
	if m, ok := charsetEncodings[charset]; ok {
		rr := transform.NewReader(input, m.NewDecoder())
		return rr, nil
	}
	charset = charsetAliases[charset]
	if m, ok := charsetEncodings[charset]; ok {
		rr := transform.NewReader(input, m.NewDecoder())
		return rr, nil
	}
	return input, nil
}

var charsetEncodings = map[string]encoding.Encoding{
	// ISO character sets
	"iso-8859-1":  charmap.ISO8859_1,
	"iso-8859-2":  charmap.ISO8859_2,
	"iso-8859-3":  charmap.ISO8859_3,
	"iso-8859-4":  charmap.ISO8859_4,
	"iso-8859-5":  charmap.ISO8859_5,
	"iso-8859-6":  charmap.ISO8859_6,
	"iso-8859-7":  charmap.ISO8859_7,
	"iso-8859-8":  charmap.ISO8859_8,
	"iso-8859-9":  charmap.ISO8859_9,
	"iso-8859-10": charmap.ISO8859_10,
	"iso-8859-13": charmap.ISO8859_13,
	"iso-8859-14": charmap.ISO8859_14,
	"iso-8859-15": charmap.ISO8859_15,
	"iso-8859-16": charmap.ISO8859_16,

	// Windows character sets
	"windows-1250": charmap.Windows1250,
	"windows-1251": charmap.Windows1251,
	"windows-1252": charmap.Windows1252,
	"windows-1253": charmap.Windows1253,
	"windows-1254": charmap.Windows1254,
	"windows-1255": charmap.Windows1255,
	"windows-1256": charmap.Windows1256,
	"windows-1257": charmap.Windows1257,
	"windows-1258": charmap.Windows1258,
	"windows-874":  charmap.Windows874,

	// DOS character sets
	"ibm437":    charmap.CodePage437,
	"ibm850":    charmap.CodePage850,
	"ibm852":    charmap.CodePage852,
	"ibm855":    charmap.CodePage855,
	"ibm858":    charmap.CodePage858,
	"ibm866":    charmap.CodePage866,
	"koi8r":     charmap.KOI8R,
	"koi8u":     charmap.KOI8U,
	"macintosh": charmap.Macintosh,

	// Japanese character sets
	"shift_jis":   japanese.ShiftJIS,
	"shift-jis":   japanese.ShiftJIS,
	"sjis":        japanese.ShiftJIS,
	"euc-jp":      japanese.EUCJP,
	"eucjp":       japanese.EUCJP,
	"iso-2022-jp": japanese.ISO2022JP,
	"iso2022jp":   japanese.ISO2022JP,

	// Korean character sets
	"euc-kr": korean.EUCKR,
	"euckr":  korean.EUCKR,

	// Chinese character sets
	"gb2312":  simplifiedchinese.GB18030, // GB18030 is a superset of GB2312
	"gbk":     simplifiedchinese.GBK,
	"gb18030": simplifiedchinese.GB18030,
	"big5":    traditionalchinese.Big5,
	"big-5":   traditionalchinese.Big5,

	// Unicode encodings
	"utf-8":    unicode.UTF8,
	"utf-16be": unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
	"utf-16le": unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf-16":   unicode.UTF16(unicode.LittleEndian, unicode.UseBOM),
}

// Alias mappings for non-standard charset names
var charsetAliases = map[string]string{
	"ascii":      "iso-8859-1", // ASCII is a subset of ISO-8859-1
	"us-ascii":   "iso-8859-1",
	"latin1":     "iso-8859-1",
	"latin2":     "iso-8859-2",
	"latin3":     "iso-8859-3",
	"latin4":     "iso-8859-4",
	"latin5":     "iso-8859-9",
	"latin6":     "iso-8859-10",
	"latin7":     "iso-8859-13",
	"latin8":     "iso-8859-14",
	"latin9":     "iso-8859-15",
	"latin10":    "iso-8859-16",
	"cp1250":     "windows-1250",
	"cp1251":     "windows-1251",
	"cp1252":     "windows-1252",
	"cp1253":     "windows-1253",
	"cp1254":     "windows-1254",
	"cp1255":     "windows-1255",
	"cp1256":     "windows-1256",
	"cp1257":     "windows-1257",
	"cp1258":     "windows-1258",
	"cp874":      "windows-874",
	"ms874":      "windows-874",
	"tis-620":    "windows-874",
	"ms-ansi":    "windows-1252",
	"ms_kanji":   "shift-jis",
	"csshiftjis": "shift-jis",
	"x-sjis":     "shift-jis",
	"ms932":      "shift-jis",
	"5601":       "euc-kr",
	"ks_c_5601":  "euc-kr",
	"ansi936":    "gb2312",
	"cp936":      "gbk",
	"ms936":      "gbk",
	"ansi950":    "big5",
	"cp950":      "big5",
	"koi8-r":     "koi8r",
	"koi8-u":     "koi8u",
}
