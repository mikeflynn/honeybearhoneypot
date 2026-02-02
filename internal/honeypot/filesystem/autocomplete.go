package filesystem

import (
	"strings"
)

// AutoComplete attempts to complete the last token in the line
func AutoComplete(currentNode *Node, line string) (string, []string) {
	// 1. Identify the word being completed
	// We preserve trailing spaces to know if we are starting a new word
	parts := strings.Split(line, " ")

	// Check if the last part is empty (trailing space)
	var toComplete string
	var preceeding string

	if len(line) > 0 && line[len(line)-1] == ' ' {
		toComplete = ""
		preceeding = line
	} else {
		toComplete = parts[len(parts)-1]
		preceeding = line[:len(line)-len(toComplete)]
	}

	// 2. Determine context (command vs argument)
	// If preceeding is empty, it's the first word -> command
	isCommand := strings.TrimSpace(preceeding) == ""

	var matches []string

	// 3. Find matches
	if isCommand {
		// Search in SystemPath (PATH) and current dir

		// Current dir
		if currentNode != nil {
			for _, child := range currentNode.Children {
				if strings.HasPrefix(child.Name, toComplete) && !child.IsCloaked() {
					name := child.Name
					if child.IsDirectory() {
						name += "/"
					} else {
						name += " "
					}
					matches = append(matches, name)
				}
			}
		}

		// PATH
		for _, p := range SystemPath {
			dirNode, err := GetNodeByPath(SystemRoot, p)
			if err == nil && dirNode != nil {
				for _, child := range dirNode.Children {
					if strings.HasPrefix(child.Name, toComplete) && !child.IsCloaked() {
						name := child.Name
						if child.IsDirectory() {
							name += "/"
						} else {
							name += " "
						}
						matches = append(matches, name)
					}
				}
			}
		}
	} else {
		// Search for file/dir relative to current node or absolute

		// Split toComplete into dir and file prefix
		lastSlash := strings.LastIndex(toComplete, "/")
		var dir string
		var prefix string

		if lastSlash != -1 {
			dir = toComplete[:lastSlash+1]
			prefix = toComplete[lastSlash+1:]
		} else {
			dir = "" // Relative to current
			prefix = toComplete
		}

		// Resolve directory
		var targetDir *Node
		var err error
		if dir == "" {
			targetDir = currentNode
		} else {
			targetDir, err = GetNodeByPath(currentNode, dir)
		}

		if err == nil && targetDir != nil && targetDir.IsDirectory() {
			for _, child := range targetDir.Children {
				if strings.HasPrefix(child.Name, prefix) && !child.IsCloaked() {
					name := child.Name
					if child.IsDirectory() {
						name += "/"
					} else {
						// Only add space if it's not a directory?
						// Usually for arguments, if it's a file, we might want a space.
						name += " "
					}
					matches = append(matches, name)
				}
			}
		}
	}

	// Deduplicate matches
	uniqueMatches := make(map[string]bool)
	var cleanMatches []string
	for _, m := range matches {
		if !uniqueMatches[m] {
			uniqueMatches[m] = true
			cleanMatches = append(cleanMatches, m)
		}
	}
	matches = cleanMatches

	if len(matches) == 0 {
		return line, nil
	}

	// 4. Find common prefix
	common := matches[0]
	for _, m := range matches[1:] {
		for !strings.HasPrefix(m, common) {
			common = common[:len(common)-1]
		}
	}

	// 5. Construct full line
	var completion string
	if isCommand {
		completion = preceeding + common
	} else {
		// Re-attach directory part
		lastSlash := strings.LastIndex(toComplete, "/")
		if lastSlash != -1 {
			completion = preceeding + toComplete[:lastSlash+1] + common
		} else {
			completion = preceeding + common
		}
	}

	return completion, matches
}
