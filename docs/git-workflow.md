# uniShell Git Workflow Cheat Sheet

This document describes the standard Git workflow for developing uniShell:

- Create a local branch.
- Push the branch to the remote with the same name.
- Work and push changes normally.
- Merge the branch into `main` **without squashing**, preserving commit history.
- Delete the branch locally and remotely after merging.
- Create a version tag to trigger a release.

The examples use `main` as the primary branch and `origin` as the GitLab remote.

---

## 1. Start From an Up-to-Date `main`

Before starting new work, make sure the local `main` is current:

```bash
git switch main
git pull --ff-only origin main
```

`--ff-only` prevents Git from silently creating an unexpected merge commit while updating your local `main`.

---

# 2. Create a New Branch Locally

Create and immediately switch to a new branch:

```bash
git switch -c <branch-name>
```

Example:

```bash
git switch -c feature/add-new-tool
```

Verify:

```bash
git branch --show-current
```

Expected:

```text
feature/add-new-tool
```

### Branch naming examples

```text
feature/add-new-tool
feature/improve-shell
fix/session-cleanup
fix/build-permissions
refactor/tool-acquisition
docs/update-installation
ci/improve-release-pipeline
```

---

# 3. Push the New Branch to the Remote

For a newly created branch, push it and establish the upstream tracking relationship:

```bash
git push -u origin <branch-name>
```

Example:

```bash
git push -u origin feature/add-new-tool
```

After this, the local branch tracks the remote branch:

```text
feature/add-new-tool -> origin/feature/add-new-tool
```

The `-u` option only needs to be used when establishing the upstream relationship for the first time.

After that, normal pushes can simply use:

```bash
git push
```

---

# 4. Work on the Branch

Make the required changes, then inspect them:

```bash
git status
git diff
```

Stage the intended changes:

```bash
git add <files>
```

Or, when appropriate:

```bash
git add .
```

Commit using the project's Conventional Commit style:

```bash
git commit -m "<type>: <description>"
```

Examples:

```bash
git commit -m "feat(tools): add fd acquisition"
git commit -m "fix(shell): preserve runtime environment"
git commit -m "refactor(build): simplify profile handling"
git commit -m "docs: update installation instructions"
```

Push the branch:

```bash
git push
```

Repeat the edit → validate → commit → push cycle as needed.

---

# 5. Merge the Branch Into `main`

## 5.1 Update the Branch Before Merging

Before merging, update your local `main`:

```bash
git switch main
git pull --ff-only origin main
```

Then switch back to the work branch:

```bash
git switch <branch-name>
```

If `main` has advanced and the branch needs to be brought up to date, merge `main` into the work branch:

```bash
git merge main
```

Resolve any conflicts if necessary, validate the result, commit the merge if Git requires one, and push:

```bash
git push
```

---

## 5.2 Merge Into `main Without Squashing

Switch to `main`:

```bash
git switch main
```

Update it:

```bash
git pull --ff-only origin main
```

Merge the work branch:

```bash
git merge --no-ff <branch-name>
```

Example:

```bash
git merge --no-ff feature/add-new-tool
```

### Why `--no-ff`?

`--no-ff` creates an explicit merge commit even when Git could perform a fast-forward merge.

This preserves the branch boundary in the history:

```text
main
  |
  |       A---B---C
  |      /         \
  +-----+-----------M
```

Instead of flattening the work into `main`:

```text
A---B---C
```

The individual commits remain intact, and the merge itself is recorded.

**Do not use ****`git merge --squash`** when the goal is to preserve the complete branch history.

---

## 5.3 Push the Updated `main`

After a successful merge:

```bash
git push origin main
```

At this point the work branch's complete history is part of `main`.

Verify:

```bash
git log --oneline --graph --decorate -n 20
```

---

# 6. Delete the Branch After Merging

Once the merge has been successfully pushed to `main`, the work branch is no longer needed.

## Delete the Local Branch

Switch to `main` first:

```bash
git switch main
```

Delete the merged local branch:

```bash
git branch -d <branch-name>
```

Example:

```bash
git branch -d feature/add-new-tool
```

The lowercase `-d` is intentional: Git will refuse to delete the branch if it believes the branch contains unmerged commits.

If Git reports that the branch is not fully merged, **do not immediately use ****`-D`**. Investigate first.

---

## Delete the Remote Branch

Delete the corresponding branch from the remote:

```bash
git push origin --delete <branch-name>
```

Example:

```bash
git push origin --delete feature/add-new-tool
```

Verify the remote-tracking references:

```bash
git fetch --prune origin
```

Then:

```bash
git branch -a
```

The deleted branch should no longer appear.

---

# 7. Complete Branch Lifecycle

The complete normal workflow is therefore:

```bash
# Start from current main
git switch main
git pull --ff-only origin main

# Create branch
git switch -c feature/my-change

# Push new branch
git push -u origin feature/my-change

# Work
git add <files>
git commit -m "feat: implement my change"
git push

# Finish work
git switch main
git pull --ff-only origin main

# Merge without squashing
git merge --no-ff feature/my-change

# Push main
git push origin main

# Delete local branch
git branch -d feature/my-change

# Delete remote branch
git push origin --delete feature/my-change

# Clean stale remote-tracking references
git fetch --prune origin
```

---

# 8. Create a Release Tag

uniShell releases are created from Git tags.

The release version uses the `vX.Y.Z` convention:

```text
v1.0.0
v1.0.1
v1.1.0
v2.0.0
```

The release pipelines are triggered by tags.

---

## 8.1 Make Sure `main` Is Clean

Before creating a release:

```bash
git switch main
git status
```

The working tree should be clean.

Then update `main`:

```bash
git pull --ff-only origin main
```

Verify the commit that will be released:

```bash
git log -1 --oneline
```

---

# 9. Create an Annotated Release Tag

Create an annotated tag:

```bash
git tag -a vX.Y.Z -m "uniShell vX.Y.Z"
```

Example:

```bash
git tag -a v1.0.1 -m "uniShell v1.0.1"
```

Verify the tag:

```bash
git show v1.0.1
```

You can also verify that the tag points to the current `main` commit:

```bash
git rev-parse v1.0.1^{commit}
git rev-parse HEAD
```

The two commit IDs should match when the tag was created from the current `HEAD`.

---

# 10. Push the Release Tag

Push the tag to the remote:

```bash
git push origin vX.Y.Z
```

Example:

```bash
git push origin v1.0.1
```

Or push the tag explicitly after creating it:

```bash
git push origin v1.0.1
```

Once the tag reaches the remote, the release CI pipeline is triggered.

---

# 11. Release Flow

The normal release sequence is:

```text
main
  |
  +--- feature/fix branches
  |
  +--- merge completed work
  |
  v
main
  |
  +--- create annotated vX.Y.Z tag
  |
  v
remote tag
  |
  +-------------------+
  |                   |
  v                   v
GitLab CI          GitHub Actions
  |                   |
  v                   v
Build binaries     Build binaries
  |                   |
  v                   v
Prepare assets     Prepare assets
  |                   |
  v                   v
GitLab Release     GitHub Release
```

The release tag identifies the exact source revision from which the release binaries are built.

---

# 12. Verify the Release Tag

After pushing the tag:

```bash
git ls-remote --tags origin vX.Y.Z
```

Example:

```bash
git ls-remote --tags origin v1.0.1
```

You can also inspect local tags:

```bash
git tag --list 'v*' --sort=-version:refname
```

---

# 13. Important Release Rules

### Release from `main`

Create release tags from the intended `main` commit:

```bash
git switch main
git pull --ff-only origin main
git tag -a vX.Y.Z -m "uniShell vX.Y.Z"
git push origin vX.Y.Z
```

### Use annotated tags

Prefer:

```bash
git tag -a v1.0.1 -m "uniShell v1.0.1"
```

over a lightweight tag:

```bash
git tag v1.0.1
```

### Do not reuse a release tag

Once a version has been released, treat its tag as immutable.

Do **not** move an existing tag to another commit to replace a release.

If a new release is required, increment the version:

```text
v1.0.0
v1.0.1
v1.0.2
```

rather than moving `v1.0.0`.

### Release version format

Use:

```text
vMAJOR.MINOR.PATCH
```

Examples:

```text
v1.0.0
v1.0.1
v1.1.0
v2.0.0
```

---

# 14. Quick Reference

## New branch

```bash
git switch main
git pull --ff-only origin main
git switch -c <branch>
git push -u origin <branch>
```

## Commit and push work

```bash
git add <files>
git commit -m "<type>: <description>"
git push
```

## Merge without squashing

```bash
git switch main
git pull --ff-only origin main
git merge --no-ff <branch>
git push origin main
```

## Delete merged branch

```bash
git branch -d <branch>
git push origin --delete <branch>
git fetch --prune origin
```

## Create release

```bash
git switch main
git pull --ff-only origin main
git status
git tag -a vX.Y.Z -m "uniShell vX.Y.Z"
git show vX.Y.Z
git push origin vX.Y.Z
```

## Inspect history

```bash
git log --oneline --graph --decorate --all
```

## Inspect branches

```bash
git branch -a
```

## Inspect tags

```bash
git tag --list 'v*' --sort=-version:refnameEnumerating objects: 7, done.
Counting objects: 100% (7/7), done.
Delta compression using up to 12 threads
Compressing objects: 100% (4/4), done.
Writing objects: 100% (4/4), 481 bytes | 481.00 KiB/s, done.
Total 4 (delta 2), reused 0 (delta 0), pack-reused 0 (from 0)
To gitlab.com:mainops/uniShell.git
   9e86213..5863581  HEAD -> mainEnumerating objects: 7, done.
Counting objects: 100% (7/7), done.
Delta compression using up to 12 threads
Compressing objects: 100% (4/4), done.
Writing objects: 100% (4/4), 481 bytes | 481.00 KiB/s, done.
Total 4 (delta 2), reused 0 (delta 0), pack-reused 0 (from 0)
To gitlab.com:mainops/uniShell.git
   9e86213..5863581  HEAD -> main
```