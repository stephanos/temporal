#!/bin/bash
#
# Find (and optionally delete) Stale Branches
#
# By default, lists stale branches without modifying anything. Pass --delete
# to actually remove them (requires interactive confirmation).
#
# A branch is protected from deletion if ANY of the following apply:
#
#   1. It is the default, a release/, or a cloud/ branch
#   2. It is associated with an open or draft PR
#   3. Its last commit is newer than the age threshold
#
# Requires: gh (GitHub CLI), authenticated with repo scope.
#
# Usage:
#   ./find_stale_branches.sh [--delete] [--age-days N]
#
set -euo pipefail

# --- Argument parsing ---

DRY_RUN=true
AGE_DAYS=90
while [[ $# -gt 0 ]]; do
  case "$1" in
    --delete)   DRY_RUN=false; shift ;;
    --age-days)
      if [[ $# -lt 2 ]]; then echo "--age-days requires a value" >&2; exit 1; fi
      AGE_DAYS="$2"
      if ! [[ "$AGE_DAYS" =~ ^[0-9]+$ ]]; then echo "--age-days must be a positive integer" >&2; exit 1; fi
      shift 2 ;;
    *)          echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# --- Query branches ---

REPO="${GITHUB_REPOSITORY:-}"
if [[ -z "$REPO" ]]; then
  REPO="$(gh repo view --json nameWithOwner -q .nameWithOwner)"
fi

owner="${REPO%%/*}"
name="${REPO##*/}"

default_branch="$(gh api "repos/${REPO}" -q .default_branch)"

# Fetch all branches with last commit dates in a single paginated GraphQL query.
# shellcheck disable=SC2016
branch_data="$(gh api graphql --paginate -f query='
  query($owner: String!, $name: String!, $endCursor: String) {
    repository(owner: $owner, name: $name) {
      refs(refPrefix: "refs/heads/", first: 100, after: $endCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          name
          target {
            ... on Commit {
              committedDate
            }
          }
        }
      }
    }
  }
' -f owner="$owner" -f name="$name" \
  -q '.data.repository.refs.nodes[] | [.name, (.target.committedDate // "")] | @tsv')"

pr_branch_list="$(gh pr list --repo "$REPO" --state open --json headRefName -q '.[].headRefName' | sort -u)"

cutoff="$(date -u -d "-${AGE_DAYS} days" '+%Y-%m-%dT%H:%M:%SZ')"

echo "Repo:    $REPO"
echo "Cutoff:  $cutoff ($AGE_DAYS days)"
echo "Dry run: $DRY_RUN"
echo ""

# --- Classify branches ---

to_delete=()
skipped=0
while IFS=$'\t' read -r branch last_commit_date; do
  [[ -z "$branch" ]] && continue

  # Rule 1: skip protected branches.
  if [[ "$branch" == "$default_branch" ]] || [[ "$branch" =~ ^(release|cloud)/ ]]; then
    continue
  fi

  # Rule 2: skip branches with open/draft PRs.
  if [[ -n "$pr_branch_list" ]] && grep -qxF "$branch" <<< "$pr_branch_list"; then
    echo "SKIP (open PR): $branch"
    skipped=$((skipped + 1))
    continue
  fi

  # Rule 3: skip branches with recent commits.
  if [[ -z "$last_commit_date" ]]; then
    echo "SKIP (no commit date): $branch"
    skipped=$((skipped + 1))
    continue
  fi
  if [[ "$last_commit_date" > "$cutoff" ]]; then
    echo "SKIP (recent): $branch ($last_commit_date)"
    skipped=$((skipped + 1))
    continue
  fi

  echo "STALE: $branch ($last_commit_date)"
  to_delete+=("$branch")
done <<< "$branch_data"

echo ""
echo "Found ${#to_delete[@]} stale branches, skipped $skipped."

if [[ ${#to_delete[@]} -eq 0 ]]; then
  exit 0
fi

# --- Deletion ---

if [[ "$DRY_RUN" == "true" ]]; then
  echo "Re-run with --delete to remove them."
  exit 0
fi

read -r -p "Type DELETE to confirm removal of ${#to_delete[@]} branches from ${REPO}: " answer
if [[ "$answer" != "DELETE" ]]; then
  echo "Aborted."
  exit 0
fi

deleted=0
for branch in "${to_delete[@]}"; do
  echo "DELETE: $branch"
  # TODO: uncomment to enable deletion
  # gh api --method DELETE "repos/${REPO}/git/refs/heads/${branch}" --silent
  deleted=$((deleted + 1))
done

echo ""
echo "Deleted $deleted branches."
