package handlers

import (
	"SimpleGit/models"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

type CommitInfo struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type Diff struct {
	Path      string      `json:"path"`
	Additions int         `json:"additions"`
	Deletions int         `json:"deletions"`
	OldPath   string      `json:"old_path,omitempty"`
	IsDeleted bool        `json:"is_deleted"`
	IsNew     bool        `json:"is_new"`
	Patches   []PatchInfo `json:"patches"`
}

type PatchInfo struct {
	Content            string `json:"content"`
	HighlightedContent string `json:"highlighted_content"`
	Type               string `json:"type"`
	OldNum             int    `json:"old_num"`
	NewNum             int    `json:"new_num"`
}

/*
func prettyPrintDiff(diff Diff) {
	fmt.Printf("\n=== File: %s ===\n", diff.Path)
	if diff.IsNew {
		fmt.Println("[NEW FILE]")
	}
	if diff.IsDeleted {
		fmt.Println("[DELETED]")
	}
	fmt.Printf("Changes: +%d -%d\n", diff.Additions, diff.Deletions)

	fmt.Println("\nPatches:")
	for i, patch := range diff.Patches {
		fmt.Printf("\n--- Patch %d ---\n", i+1)
		fmt.Printf("Type: %s\n", patch.Type)
		fmt.Printf("Content:\n%s\n", patch.Content)
		fmt.Printf("-------------\n")
	}
	fmt.Println("==================")
}
*/

func (s *Server) handleViewCommit(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		models.HandleError(w, r, models.NewBadRequestError("Invalid commit path"))
		return
	}

	repoName := parts[1]
	commitHash := parts[2]

	repo, ok := s.Repos[repoName]
	if !ok {
		models.HandleError(w, r, models.NewNotFoundError("Repository not found"))
		return
	}

	gitRepo, err := repo.Git()
	if err != nil {
		models.HandleError(w, r, models.NewGitError("Failed to open repository", err))
		return
	}

	hash := plumbing.NewHash(commitHash)
	commit, err := gitRepo.CommitObject(hash)
	if err != nil {
		models.HandleError(w, r, models.NewNotFoundError("Commit not found"))
		return
	}

	parentCommit, err := commit.Parent(0)
	if err != nil {
		models.HandleError(w, r, models.NewGitError("Failed to get parent commit", err))
		return
	}

	var diffs []Diff
	if parentCommit != nil {
		//Get the trees for comparison
		parentTree, err := parentCommit.Tree()
		if err != nil {
			models.HandleError(w, r, models.NewGitError("Failed to get parent tree", err))
			return
		}

		currentTree, err := commit.Tree()
		if err != nil {
			models.HandleError(w, r, models.NewGitError("Failed to get current tree", err))
			return
		}

		changes, err := currentTree.Diff(parentTree)
		if err != nil {
			models.HandleError(w, r, models.NewGitError("Failed to get diff", err))
			return
		}

		// Pre-allocate the diffs slice
		diffs = make([]Diff, 0, len(changes))

		for _, change := range changes {
			from, to := change.From, change.To
			diff := Diff{
				Path:      to.Name,
				OldPath:   from.Name,
				IsDeleted: to.Name == "",
				IsNew:     from.Name == "",
			}

			patch, err := change.Patch()
			if err != nil {
				continue
			}

			// Process file stats
			// NOTE: go-git's diff shows changes from current -> parent
			// but we want to show parent -> current, so we swap Addition/Deletion
			for _, fileStat := range patch.Stats() {
				diff.Additions += fileStat.Deletion // Deletion in parent means Addition in current
				diff.Deletions += fileStat.Addition // Addition in parent means Deletion in current
			}

			// Pre-allocate patches slice
			diff.Patches = make([]PatchInfo, 0, len(patch.FilePatches())*10) // rough estimate

			for _, p := range patch.FilePatches() {
				oldLineNum := 0
				newLineNum := 0

				// Track the last printed line numbers to add spacing between chunks
				lastOldNum := 0
				lastNewNum := 0

				for _, chunk := range p.Chunks() {
					lines := strings.Split(strings.TrimRight(chunk.Content(), "\n"), "\n")

					// For context chunks, only show 3 lines before and after changes
					if chunk.Type() == 0 { // Equal/Context
						if lastOldNum > 0 { // After a change
							// Show only 3 lines after a change
							if len(lines) > 3 {
								lines = lines[:3]
							}
						} else { // Before a change
							// Show only 3 lines before a change
							if len(lines) > 3 {
								oldLineNum += len(lines) - 3
								newLineNum += len(lines) - 3
								lines = lines[len(lines)-3:]
							}
						}
					}

					// If there's a gap between chunks, add a separator line
					if (lastOldNum > 0 && oldLineNum > lastOldNum+1) ||
						(lastNewNum > 0 && newLineNum > lastNewNum+1) {
						diff.Patches = append(diff.Patches, PatchInfo{
							Content: "...",
							Type:    "separator",
							OldNum:  0,
							NewNum:  0,
						})
					}

					// Process each line in the chunk
					for _, line := range lines {
						patchInfo := PatchInfo{
							Content: line,
						}

						if line == "..." {
							patchInfo.Type = "separator"
							patchInfo.Content = line
							patchInfo.HighlightedContent = line
						} else {
							// Get file extension for language detection
							ext := filepath.Ext(diff.Path)
							if ext != "" {
								ext = ext[1:] // Remove the leading dot
							}

							// Highlight the content
							result, err := s.tsService.Highlight(line, ext, diff.Path)
							if err != nil {
								// On error, fall back to unhighlighted content
								patchInfo.HighlightedContent = line
							} else {
								patchInfo.HighlightedContent = result.Highlighted
							}
						}

						// Set line numbers and type based on chunk type
						// NOTE: go-git's chunks are from the perspective of going from current -> parent
						// but we want to show parent -> current, so we swap the meaning of Add/Delete
						switch chunk.Type() {
						case 0: // Equal - no change
							oldLineNum++
							newLineNum++
							patchInfo.Type = "context"
							patchInfo.OldNum = oldLineNum
							patchInfo.NewNum = newLineNum
						case 1: // Add in parent means it was deleted in current
							newLineNum++
							patchInfo.Type = "deletion"
							patchInfo.OldNum = 0
							patchInfo.NewNum = newLineNum
						case 2: // Delete in parent means it was added in current
							oldLineNum++
							patchInfo.Type = "addition"
							patchInfo.OldNum = oldLineNum
							patchInfo.NewNum = 0
						}

						diff.Patches = append(diff.Patches, patchInfo)
					}

					lastOldNum = oldLineNum
					lastNewNum = newLineNum
				}
			}

			diffs = append(diffs, diff)
		}
	}

	data := map[string]interface{}{
		"Repo": repo,
		"Commit": CommitInfo{
			Hash:      commit.Hash.String(),
			Author:    commit.Author.Name,
			Email:     commit.Author.Email,
			Message:   commit.Message,
			Timestamp: commit.Author.When,
		},
		"Diffs": diffs,
	}

	if err := s.tmpl.ExecuteTemplate(w, "commit.html", s.addCommonData(r, data)); err != nil {
		models.HandleError(w, r, models.NewInternalError("Failed to render template").WithError(err))
		return
	}
}
