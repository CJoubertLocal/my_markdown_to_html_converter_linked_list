package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

/*
TODO: Alternative would be to simply seek characters are replace the strings.
E.g. find '```' and then find the first instance of '```' afterwards.
*/

var firstLinkedListNode *linkedListNode
var currentLinkedListNode *linkedListNode

var bytesReadIn []byte
var byteReader *bytes.Reader

var thereIsACodeBlockToClose = false
var numberOfOpeningParagraphTags = 0
var numberOfClosingParagraphTags = 0
var footnoteCount = 1
var footnoteEndOfPagecount = 1
var inFootnotesSection = false

var htmlEntitiesMap = map[rune]string{
	'\'': "&apos;",
	'<':  "&lt;",
	'>':  "&gt;",
	'"':  "&quot;",
	'-':  "&ndash;",
}

type linkedListNode struct {
	previousNode *linkedListNode
	nextNode     *linkedListNode
	content      string
}

func (ln *linkedListNode) AppendNewLinkedListNode(contents string) {
	newNode := linkedListNode{
		previousNode: currentLinkedListNode,
		content:      contents,
	}

	currentLinkedListNode.nextNode = &newNode
	currentLinkedListNode = &newNode
}

func (ln *linkedListNode) ReplaceContentsOfCurrentNode(newContents string) {
	ln.content = newContents
}

func (ln *linkedListNode) JoinContentsOfNodes() string {
	if ln.nextNode != nil {
		res := ln.content + ln.nextNode.JoinContentsOfNodes()
		return res
	}

	return ln.content
}

func (ln *linkedListNode) FindLastNodeWithContentAndReplaceAllFollowingNodesWithSpecificContents(contentToFind string, contentToReplace string, contentToReplaceWith string) {
	anchorNode := ln

	for anchorNode.content != contentToFind {
		if anchorNode.content == contentToReplace {
			if contentToReplace == "<p>" {
				numberOfOpeningParagraphTags--
			}
			if contentToReplace == "</p>" {
				numberOfClosingParagraphTags--
			}
			anchorNode.content = contentToReplaceWith
		}
		anchorNode = anchorNode.previousNode
	}
}

func main() {
	pathName := os.Args
	res := convertMarkdownFileToBlogHTML(pathName[1])
	saveToFile(res)
}

func convertMarkdownFileToBlogHTML(fileNameAndPath string) string {
	fmt.Println(fileNameAndPath)
	// https://pkg.go.dev/os#ReadDir
	var err error

	bytesReadIn, err = os.ReadFile(fileNameAndPath)
	if err != nil {
		log.Fatal("unable to find file:", err)
	}

	byteReader = bytes.NewReader(bytesReadIn)
	firstLinkedListNode = &linkedListNode{}
	currentLinkedListNode = firstLinkedListNode
	numberOfOpeningParagraphTags = 0
	numberOfClosingParagraphTags = 0
	thereIsACodeBlockToClose = false
	inFootnotesSection = false
	footnoteCount = 1
	footnoteEndOfPagecount = 1

	currentLinkedListNode.AppendNewLinkedListNode("<div class=\"content\">")

	for {
		newR, _, err := byteReader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("unable to read rune:", err)
		}
		runeMatcher(newR)
	}

	if numberOfOpeningParagraphTags > numberOfClosingParagraphTags {
		currentLinkedListNode.AppendNewLinkedListNode("</p>")
	}

	currentLinkedListNode.AppendNewLinkedListNode("</div>")

	res := strings.ReplaceAll(firstLinkedListNode.JoinContentsOfNodes(), "<p></p>", "")
	return res
}

// https://gobyexample.com/writing-files
func saveToFile(res string) {
	f, err := os.Create("../tmp/html_out.html")
	if err != nil {
		log.Fatal("unable to create file:", err)
	}
	defer f.Close()

	numBytesWritten, err := f.WriteString(res)
	if err != nil {
		log.Fatal("error when writing to file:", err)
	}
	fmt.Printf("wrote %d bytes to file", numBytesWritten)

	f.Sync()
}

/*
Assumption:
All posts start with a '# Header', and therefore should not start with '<p>'
All new line characters / carriage returns should be handled by this interpreter if there is the
possibility of a zero-length line.
*/
func runeMatcher(r rune) {
	switch r {
	case '`':
		insertCodeBlock()
	case '\'':
		addHTMLEntity(r)
	case '<':
		addHTMLEntity(r)
	case '>':
		addHTMLEntity(r)
	case '"':
		addHTMLEntity(r)
	case '-':
		addHTMLEntity(r)
	case '*':
		determineIfItalicsOrBold()
	case '#':
		addHeader()
	case '[':
		checkIfFootnoteShouldBeAdded()
	case '|':
		checkIfTableShouldBeAdded()
	case '\r':
		checkIfParagraphTagsShouldBeAdded() // Carriage return
	default:
		addRuneAsIs(r)
	}
}

func checkIfNextRuneIs(r rune) (bool, bool) {
	nextR, _, err := byteReader.ReadRune()
	if err == io.EOF {
		return true, false
	}
	if err != nil {
		log.Fatal("unable to read rune:", err)
	}
	return false, nextR == r
}

func unreadRune() {
	err := byteReader.UnreadRune()
	if err != nil {
		log.Fatal("unable to unread rune:", err)
	}
}

func insertCodeBlock() {
	isEOF, isRune := checkIfNextRuneIs('`')
	if isEOF {
		return
	}
	if !isRune {
		unreadRune()
		inlineCodeBlock()
		return
	}

	isEOF, isRune = checkIfNextRuneIs('`')
	if isEOF {
		return
	}
	if !isRune {
		log.Fatal("malformed code block in input")
	} else {
		multilineCodeBlock()
	}
}

func inlineCodeBlock() {
	var err error
	var newR rune

	currentLinkedListNode.AppendNewLinkedListNode("<code>")
	for {
		newR, _, err = byteReader.ReadRune()
		if err != nil {
			log.Fatal("unable to read rune in inline code block:", err)
		}
		if newR == '`' {
			break
		}
		runeMatcher(newR)
	}

	currentLinkedListNode.AppendNewLinkedListNode("</code>")
}

func multilineCodeBlock() {
	if thereIsACodeBlockToClose {
		currentLinkedListNode.FindLastNodeWithContentAndReplaceAllFollowingNodesWithSpecificContents("<code>", "</p><p>", "")
		currentLinkedListNode.FindLastNodeWithContentAndReplaceAllFollowingNodesWithSpecificContents("<code>", "<p>", "")
		currentLinkedListNode.FindLastNodeWithContentAndReplaceAllFollowingNodesWithSpecificContents("<code>", "</p>", "")
		currentLinkedListNode.AppendNewLinkedListNode("</code>")
		currentLinkedListNode.AppendNewLinkedListNode("</pre>")
		thereIsACodeBlockToClose = false
		return
	}

	for {
		curRune, _, err := byteReader.ReadRune()
		if err != nil {
			log.Fatal("unable to read rune:", err)
		}
		if curRune == '\r' {
			isEOF, isRune := checkIfNextRuneIs('\n')
			if isEOF {
				return
			}
			if !isRune {
				log.Fatal("inserting code block: \\r is not followed by \\n")
			} else {
				break
			}
		}
	}

	currentLinkedListNode.AppendNewLinkedListNode("<pre>")
	currentLinkedListNode.AppendNewLinkedListNode("<code>")

	thereIsACodeBlockToClose = true
}

func addHTMLEntity(r rune) {
	currentLinkedListNode.AppendNewLinkedListNode(htmlEntitiesMap[r])
}

func determineIfItalicsOrBold() {
	if thereIsACodeBlockToClose {
		currentLinkedListNode.AppendNewLinkedListNode("*")
		return
	}

	nextRune, _, err := byteReader.ReadRune()
	if err != nil {
		log.Fatal("error when reading next rune to determine if italics or bold:", err)
	}

	if nextRune == '*' {
		runeAfter, _, err := byteReader.ReadRune()
		if err != nil {
			log.Fatal("unable to read nextRune when determining if italics or bold text:", err)
		}

		// ***rest_of_file
		if runeAfter == '*' {
			currentLinkedListNode.AppendNewLinkedListNode("<i>")
			addBoldTags()
			currentLinkedListNode.AppendNewLinkedListNode("</i>")

			// Read remaining italics asterisk to avoid looping back here
			_, _, err = byteReader.ReadRune()
			if err != nil {
				log.Fatal("unable to read remaining italics asterisk:", err)
			}
			return
		}

		// **rest_of_file
		err = byteReader.UnreadRune()
		if err != nil {
			log.Fatal("unable to unread rune before converting inserting bold tags:", err)
		}
		addBoldTags()
		return
	}

	// *rest_of_file
	err = byteReader.UnreadRune()
	if err != nil {
		log.Fatal("unable to unread rune before adding italics tags:", err)
	}
	addItalicsTags()
}

func addItalicsTags() {
	currentLinkedListNode.AppendNewLinkedListNode("<i>")

	var err error
	var r rune
	for {
		r, _, err = byteReader.ReadRune()
		if err != nil {
			log.Fatal("unable to read rune when adding italics tags:", err)
		}
		if r == '*' {
			break
		}
		runeMatcher(r)
	}
	currentLinkedListNode.AppendNewLinkedListNode("</i>")
}

// Assumes that there are no italics within the bold text
func addBoldTags() {
	currentLinkedListNode.AppendNewLinkedListNode("<b>")

	var err error
	var r rune
	for {
		r, _, err = byteReader.ReadRune()
		if err != nil {
			log.Fatal("unable to read rune when adding bold tags:", err)
		}

		if r == '*' {
			nextR, _, err := byteReader.ReadRune()
			if err != nil {
				log.Fatal("unable to read next rune when adding bold tags:", err)
			}

			if nextR == '*' {
				break

			} else {
				err = byteReader.UnreadRune()
				if err != nil {
					log.Fatal("unable to unread rune when checking if bold tags are closed:", err)
				}
			}

			currentLinkedListNode.AppendNewLinkedListNode("*")
		}
		runeMatcher(r)
	}
	currentLinkedListNode.AppendNewLinkedListNode("</b>")
}

func addHeader() {
	if thereIsACodeBlockToClose || inFootnotesSection {
		currentLinkedListNode.AppendNewLinkedListNode("#")
		return
	}

	headerNumber := 1

	for {
		nextR, _, err := byteReader.ReadRune()
		if err != nil {
			log.Fatal("unable to read next rune when determining header tag:", err)
		}
		if nextR != '#' {
			err = byteReader.UnreadRune()
			if err != nil {
				log.Fatal("unable to unread rune when determining header tag:", err)
			}
			break
		}
		headerNumber++
	}

	currentLinkedListNode.AppendNewLinkedListNode("<h" + strconv.Itoa(headerNumber) + ">")

	for {
		nextR, _, err := byteReader.ReadRune()
		if err == io.EOF {
			break
		}
		if nextR == '\r' {
			unreadRune() // So that the rune matcher correctly tries to link to a paragraph.
			break
		}
		if err != nil {
			log.Fatal("unable to read rune:", err)
		}
		addRuneAsIs(nextR)
	}

	currentLinkedListNode.AppendNewLinkedListNode("</h" + strconv.Itoa(headerNumber) + ">")
}

func checkIfFootnoteShouldBeAdded() {
	curRune, _, err := byteReader.ReadRune()
	if err == io.EOF {
		return
	}
	if err != nil {
		log.Fatal("error when reading rune for possible footnote:", err)
	}
	if curRune == '^' {
		addFootnote()
		return
	}
	unreadRune()
	addNonFootnoteBetweenTwoSquareBrackets()
}

func addFootnote() {
	// Read until ']:'.
	// If there is a ':', then this is a footnote at the end of the page.
	// If not, this is an inline-footnote. Unread a rune and insert the footnote.
	for {
		curRune, _, err := byteReader.ReadRune()
		if err == io.EOF {
			break
		}
		if curRune == ']' {
			break
		}
		if err != nil {
			log.Fatal("error when reading rune:", err)
		}
	}

	isEOF, isRune := checkIfNextRuneIs(':')
	if !isRune {
		fn := strconv.Itoa(footnoteCount)
		footnoteCount++
		if !isEOF {
			unreadRune()
		}
		currentLinkedListNode.AppendNewLinkedListNode("<a id=\"footnote-anchor-" + fn + "\" href=\"#footnote-" + fn + "\">[" + fn + "]</a>")

	} else {
		inFootnotesSection = true
		fn := strconv.Itoa(footnoteEndOfPagecount)
		footnoteEndOfPagecount++
		if numberOfOpeningParagraphTags > numberOfClosingParagraphTags {
			currentLinkedListNode.AppendNewLinkedListNode("</p>")
			numberOfClosingParagraphTags++
		}
		currentLinkedListNode.AppendNewLinkedListNode("<p id=\"footnote-" + fn + "\"><a href=\"#footnote-anchor-" + fn + "\">[" + fn + "]</a>")
		numberOfOpeningParagraphTags++
	}
}

func addNonFootnoteBetweenTwoSquareBrackets() {
	currentLinkedListNode.AppendNewLinkedListNode("[")
	for {
		curRune, _, err := byteReader.ReadRune()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatal("unable to read rune:", err)
		}
		if curRune == ']' {
			addRune(']')
			break
		} else {
			addRune(curRune)
		}
	}
}

// Assumption: there is no recursive table-within-a-table feature.
// Anything within a table is plain text. <- will have to remove this assumption, to replace HTML entities.
func checkIfTableShouldBeAdded() {
	if thereIsACodeBlockToClose {
		currentLinkedListNode.AppendNewLinkedListNode("|")
		for {
			curRune, _, err := byteReader.ReadRune()
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Fatal("unable to read rune:", err)
			}
			if curRune == '\r' {
				unreadRune()
				return
			}
			addRune(curRune)
		}
	}
	// If next rune is '|', then this is a malformed table, or not a table at all.
	// E.g. it is '||' in a programming language.
	nextR, _, err := byteReader.ReadRune()
	if err == io.EOF {
		return
	}
	if err != nil {
		log.Fatal("unable to read next rune:", nextR)
	}
	if nextR == '|' {
		// "||"
		return
	}
	unreadRune()

	currentLinkedListNode.AppendNewLinkedListNode("<table class=\"table table-hover\">")
	addTableHeader()
	readPastTableBreakLine()
	addTableBody()

	currentLinkedListNode.AppendNewLinkedListNode("</table>")
}

func addTableHeader() {
	currentLinkedListNode.AppendNewLinkedListNode("<thead><tr><th scope=\"col\">")

	var tableHeadRune rune
	var err error
	for {
		tableHeadRune, _, err = byteReader.ReadRune()
		if err == io.EOF || tableHeadRune == '\n' {
			// The simplest test case of just the table header will result in malformed input if
			// io.EOF is not treated the same as '\n'.
			currentLinkedListNode.ReplaceContentsOfCurrentNode("</tr></thead>")
			return
		}
		if err != nil {
			log.Fatal("unable to read rune:", err)
		}
		if tableHeadRune == '|' {
			currentLinkedListNode.AppendNewLinkedListNode("</th>")
			currentLinkedListNode.AppendNewLinkedListNode("<th scope=\"col\">")
		} else if tableHeadRune != '\r' {
			addRune(tableHeadRune)
		}
	}
}

func readPastTableBreakLine() {
	for {
		nextR, _, err := byteReader.ReadRune()
		if err == io.EOF || nextR == '\n' {
			return
		}
		if err != nil {
			log.Fatal("unable to read rune when skipping table breakline:", err)
		}
	}
}

// Assumes that the table row functions read their carriage returns.
// Assumes that there is a blank link after a table.
func addTableBody() {
	currentLinkedListNode.AppendNewLinkedListNode("<tbody>")
	for {
		r, _, err := byteReader.ReadRune()
		if err == io.EOF {
			break
		}
		if r == '\r' {
			// Found carriage return after another carriage return, table has ended
			unreadRune()
			break
		}
		if r == '|' {
			// New table row found
			addTableBodyRow()
		}
	}
	currentLinkedListNode.AppendNewLinkedListNode("</tbody>")
}

func addTableBodyRow() {
	currentLinkedListNode.AppendNewLinkedListNode("<tr>")
	currentLinkedListNode.AppendNewLinkedListNode("<td>")

	for {
		r, _, err := byteReader.ReadRune()
		if err == io.EOF || r == '\n' {
			// Past carriage return
			break
		}
		if err != nil {
			log.Fatal("malformed table body row:", err)
		}
		if r == '|' {
			currentLinkedListNode.AppendNewLinkedListNode("</td>")
			currentLinkedListNode.AppendNewLinkedListNode("<td>")
		} else if r != '\r' {
			addRune(r)
		}
	}

	currentLinkedListNode.ReplaceContentsOfCurrentNode("</tr>")
}

func checkIfParagraphTagsShouldBeAdded() {
	isEOF, isRune := checkIfNextRuneIs('\n')
	if isEOF {
		return
	}
	if !isRune {
		log.Fatal("\\r is not followed by \\n")
	}

	if thereIsACodeBlockToClose {
		currentLinkedListNode.AppendNewLinkedListNode("\n")
		return
	}

	isEOF, isRune = checkIfNextRuneIs('\r')
	if isEOF {
		return
	}
	if !isRune {
		unreadRune()

	} else {
		checkIfNextRuneIs('\n')
		isEOF, isRune = checkIfNextRuneIs('-')
		if isEOF {
			return
		}
		if isRune {
			// List
			addUnorderedList()
			return
		}
		unreadRune()
		addParagraphTags()
	}
}

func addParagraphTags() {
	if numberOfOpeningParagraphTags == 0 {
		currentLinkedListNode.AppendNewLinkedListNode("<p>")
		numberOfOpeningParagraphTags++

	} else if numberOfOpeningParagraphTags >= numberOfClosingParagraphTags {
		currentLinkedListNode.AppendNewLinkedListNode("</p><p>")
		numberOfOpeningParagraphTags++
		numberOfClosingParagraphTags++
	}
}

func addUnorderedList() {
	currentLinkedListNode.AppendNewLinkedListNode("<p>")
	currentLinkedListNode.AppendNewLinkedListNode("<ul>")
	currentLinkedListNode.AppendNewLinkedListNode("<li>")

	numNewLinesInARow := 0
	for numNewLinesInARow < 2 {
		curRune, _, err := byteReader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("unable to read rune:", err)
		}
		if curRune == '\r' {
			currentLinkedListNode.AppendNewLinkedListNode("</li>")
			numNewLinesInARow++
		} else if curRune == '-' && numNewLinesInARow == 1 {
			numNewLinesInARow--
			currentLinkedListNode.AppendNewLinkedListNode("<li>")
		} else if curRune != '\n' {
			addRune(curRune)
		}
	}

	currentLinkedListNode.AppendNewLinkedListNode("</li>")
	currentLinkedListNode.AppendNewLinkedListNode("</ul>")
	currentLinkedListNode.AppendNewLinkedListNode("</p>")
}

// TODO: this feels redundant.
func addRune(r rune) {
	if r == '[' {
		checkIfFootnoteShouldBeAdded()
		return
	}
	for htmlR, htmlEntity := range htmlEntitiesMap {
		if htmlR == r {
			currentLinkedListNode.AppendNewLinkedListNode(htmlEntity)
			return
		}
	}
	addRuneAsIs(r)
}

func addRuneAsIs(r rune) {
	currentLinkedListNode.AppendNewLinkedListNode(string(r))
}
