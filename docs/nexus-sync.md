# uniShell Nexus Synchronization

## Purpose

`lesscache-sync.sh` synchronizes the latest uniShell release published on GitLab into the Nexus Raw repository used as the artifact distribution backend for deployed uniShell environments.

The synchronizer is intentionally conservative:

- GitLab is the release source of truth.
- `SHA256SUMS` is the authoritative artifact manifest.
- Nexus stores the synchronized release marker and release binaries.
- An already synchronized release is a true no-op.
- A newer GitLab release advances Nexus.
- An older GitLab release is refused.
- Release artifacts are verified before publication and after publication.
- Temporary downloaded artifacts are removed after execution.

The current deployed synchronizer is:

```text
$HOME/.local/state/.lesscache.sh
```

## Current Architecture

```text
GitLab Release
     |
     | latest release redirect
     v
SHA256SUMS + unishell-* binaries
     |
     | download + SHA-256 verification
     v
Local staging
$HOME/.local/state/.lesscache
     |
     | authenticated Nexus Raw upload
     v
Nexus
raw/lesscache/
     |
     +-- lesscache-full
     +-- lesscache-k8s
     +-- lesscache-slim
     +-- ...
     |
     +-- v<release>
```

The synchronizer does not use GitLab authentication. The latest public GitLab release is discovered through the public `permalink/latest` release URL.

Nexus requires authentication.

## Endpoints and Storage

### GitLab

Release discovery:

```text
https://gitlab.com/mainops/uniShell/-/releases/permalink/latest
```

Release artifacts are downloaded from the corresponding release's download area.

The release must use the version-tag format:

```text
vMAJOR.MINOR.PATCH
```

The release manifest is:

```text
SHA256SUMS
```

### Nexus

Nexus base URL:

```text
https://ntgrepo.ntgcloud.net
```

Repository:

```text
raw
```

Namespace:

```text
lesscache/
```

The resulting Nexus asset paths are therefore:

```text
lesscache/<artifact>
lesscache/<release-tag>
```

The Nexus REST API is used for release-state discovery, asset upload, asset verification, and marker cleanup.

## Authentication

Nexus credentials are read from:

```text
$HOME/.local/state/.repo
```

The file contains the Nexus credentials in the expected:

```text
username:password
```

form.

The credential path is intentionally not configurable by the synchronizer.

GitLab authentication is not required.

The credential file must not be committed to the repository or otherwise distributed with uniShell releases.

## Release Marker

The Nexus release marker is a self-contained representation of the synchronized GitLab release.

For release `v1.0.1`, the marker is:

```text
lesscache/v1.0.1
```

Its contents are the GitLab `SHA256SUMS` manifest.

The marker therefore identifies both:

1. the release version; and
    
2. the exact artifact checksums associated with that release.
    

The marker is verified after publication.

Only the current synchronized release marker should remain in Nexus after a successful update.

## Artifact Naming

GitLab release assets retain their GitLab names during download and checksum verification.

The Nexus names are derived by replacing the `unishell-` prefix with `lesscache-`.

Examples:

```text
GitLab              Nexus
------              -----
unishell-full       lesscache-full
unishell-k8s        lesscache-k8s
unishell-slim       lesscache-slim
```

The mapping is prefix-based rather than a fixed list, allowing additional `unishell-*` release artifacts to be synchronized without modifying the naming logic.

## Synchronization State Machine

The synchronizer first determines:

1. the latest GitLab release;
    
2. the current Nexus release marker.
    

It then compares the two versions.

### Initial synchronization

Condition:

```text
Nexus has no release marker
```

Behavior:

```text
GitLab latest release
        |
        v
Download SHA256SUMS
        |
        v
Discover unishell-* artifacts
        |
        v
Download artifacts
        |
        v
Verify SHA-256
        |
        v
Upload artifacts to Nexus
        |
        v
Verify Nexus artifacts
        |
        v
Publish release marker
        |
        v
Verify release marker
        |
        v
Remove older markers
        |
        v
Verify marker cleanup
```

### Same release

Condition:

```text
GitLab release == Nexus release
```

Behavior:

```text
No manifest download
No artifact download
No artifact upload
No marker upload
No marker deletion
```

The operation terminates successfully as a true no-op.

This avoids unnecessary network traffic and prevents an already synchronized release from being republished.

### Newer GitLab release

Condition:

```text
GitLab release > Nexus release
```

Behavior:

1. Download the GitLab `SHA256SUMS`.
    
2. Discover the release's `unishell-*` artifacts.
    
3. Download every listed artifact.
    
4. Verify every artifact against `SHA256SUMS`.
    
5. Upload the verified artifacts to Nexus using their `lesscache-*` names.
    
6. Verify the uploaded Nexus assets against the authoritative GitLab checksums.
    
7. Publish the new release marker containing `SHA256SUMS`.
    
8. Verify the marker.
    
9. Remove older Nexus release markers.
    
10. Verify that only the current release marker remains.
    
11. Remove the local staging artifacts.
    

Existing Nexus artifacts with the same `lesscache-*` names are overwritten when advancing to a newer release.

### Older GitLab release

Condition:

```text
GitLab release < Nexus release
```

Behavior:

```text
Refuse synchronization
```

The synchronizer terminates with a non-zero status before downloading the manifest or artifacts.

This protects Nexus from accidental rollback when GitLab's currently published release is older than the version already synchronized.

If an intentional rollback is required, it is an operator-controlled Nexus operation rather than an automatic synchronization operation.

## Manifest and Integrity

`SHA256SUMS` is treated as the authoritative manifest for the release.

Each manifest entry is expected to contain:

```text
<64-character SHA-256>  unishell-<artifact>
```

Every downloaded release artifact is checked against its manifest checksum before it can be uploaded.

After upload, the Nexus asset is downloaded or otherwise retrieved through the Nexus API and its SHA-256 checksum is compared with the expected GitLab checksum.

The release marker itself is also verified against the local `SHA256SUMS`.

This produces three integrity checks:

```text
GitLab artifact
      |
      v
local artifact checksum
      |
      v
Nexus artifact checksum

GitLab SHA256SUMS
      |
      v
Nexus release marker checksum
```

## Staging

Temporary release artifacts are stored in:

```text
$HOME/.local/state/.lesscache
```

The staging directory is cleared before preparing a new synchronization.

The synchronizer registers cleanup on termination after staging has been initialized, ensuring downloaded artifacts are removed after successful or failed synchronization.

A same-version no-op does not need to create the staging directory.

Downloaded release artifacts are therefore not retained as a local artifact cache.

## Nexus API Operations

The synchronizer uses the Nexus REST API for the following operations:

|Operation|Nexus API purpose|
|---|---|
|Release discovery|Enumerate/search assets and identify `lesscache/v*` markers|
|Artifact discovery|Locate synchronized `lesscache-*` assets|
|Artifact upload|Upload verified Raw assets|
|Artifact verification|Retrieve Nexus assets and compare checksums|
|Marker publication|Upload `SHA256SUMS` under the release tag|
|Marker verification|Retrieve the release marker and verify its checksum|
|Marker cleanup|Delete older release-marker assets|

The Nexus API specification used during implementation is retained in the project as:

```text
nexus-swagger.json
```

## Failure and Safety Behavior

The synchronization process is intentionally ordered so that destructive or state-changing operations happen only after source and artifact validation.

The normal advancement sequence is:

```text
Discover
  |
  v
Validate release
  |
  v
Download
  |
  v
Verify
  |
  v
Upload
  |
  v
Verify uploaded state
  |
  v
Publish marker
  |
  v
Verify marker
  |
  v
Delete obsolete markers
  |
  v
Verify cleanup
```

An older release is rejected before artifact staging begins.

A failed artifact checksum prevents that artifact from being published.

A failed Nexus verification prevents the synchronization from being considered successful.

Marker cleanup occurs only after the new release marker and artifacts have been successfully published and verified.

## Systemd User Scheduling

The synchronizer is deployed as:

```text
$HOME/.local/state/.lesscache.sh
```

The systemd user service is:

```text
$HOME/.config/systemd/user/lesscache-sync.service
```

The systemd user timer is:

```text
$HOME/.config/systemd/user/lesscache-sync.timer
```

The service is a `oneshot` unit and explicitly invokes Bash:

```text
/usr/bin/env bash %h/.local/state/.lesscache.sh
```

The timer runs once per day beginning at 05:00 with a randomized delay of up to three hours:

```text
OnCalendar=*-*-* 05:00:00
RandomizedDelaySec=3h
```

This distributes synchronization traffic across the 05:00–08:00 window rather than causing all deployed hosts to contact Nexus simultaneously.

The timer uses:

```text
Persistent=true
```

so a missed scheduled invocation can be handled when the user systemd manager becomes active again.

## Operational Verification

The synchronization implementation has been runtime-tested for the following states:

|Scenario|Result|
|---|---|
|Initial synchronization|Validated|
|Same-version synchronization|Validated|
|Newer release synchronization|Validated|
|Artifact checksum verification|Validated|
|Nexus artifact naming|Validated|
|Nexus artifact overwrite|Validated|
|Release marker publication|Validated|
|Release marker verification|Validated|
|Older-release refusal|Validated|
|Older-marker cleanup|Validated|
|Staging cleanup|Validated|
|systemd service execution|Validated|
|systemd timer activation|Validated|
|Randomized timer window|Validated|

The older-release test used a temporary `v999.0.0` Nexus marker while GitLab remained at `v1.0.1`. The synchronizer correctly refused synchronization with a non-zero status and did not begin artifact staging. The temporary marker was subsequently removed and the legitimate `v1.0.1` Nexus state was restored.

## Deployment Contract

A target host running the synchronizer requires:

```text
$HOME/.local/state/.lesscache.sh
$HOME/.local/state/.repo
$HOME/.config/systemd/user/lesscache-sync.service
$HOME/.config/systemd/user/lesscache-sync.timer
```

The synchronizer also requires the host to provide the command-line utilities used by the implementation, including:

```text
bash
curl
jq
sort
wc
sha256sum
awk
find
```

The Nexus credential file must be available before the synchronizer is invoked.

The systemd user manager must be available for scheduled execution.

## Future Nexus Migration

Nexus is the current artifact backend, but the project is intended to be able to migrate away from it.

The future replacement must preserve the synchronization contract rather than coupling uniShell to Nexus-specific implementation details.

The migration target currently identified is `pulp-project`.

### Required protocol invariants

The replacement backend must support:

1. Authentication suitable for unattended synchronization.
    
2. Discovery of the current synchronized release.
    
3. Publication of multiple release artifacts.
    
4. Artifact retrieval for post-publication verification.
    
5. SHA-256 integrity verification.
    
6. A release/version marker or equivalent metadata.
    
7. Replacement of artifacts when advancing to a newer release.
    
8. Removal or supersession of obsolete release markers.
    
9. Atomic enough publication semantics to prevent a partially published release from being considered current.
    
10. A machine-readable API suitable for the synchronizer.
    
11. Unattended operation from the systemd user service.
    
12. Explicit failure reporting.
    

### Required logical storage model

The backend should preserve the logical distinction between:

```text
release version
release manifest
release artifacts
```

The current Nexus representation is:

```text
lesscache/
    lesscache-full
    lesscache-k8s
    lesscache-slim
    ...
    v<release>
```

A future backend does not need to reproduce this physical layout exactly, provided it can represent the same logical state and synchronization guarantees.

### Migration requirements

Before replacing Nexus in production:

1. Document the Pulp API and authentication model.
    
2. Define the equivalent release-marker representation.
    
3. Define artifact naming and storage.
    
4. Define upload and overwrite semantics.
    
5. Define post-upload verification.
    
6. Define cleanup and rollback semantics.
    
7. Implement the backend independently from the release-discovery and checksum logic.
    
8. Run the same initial, same-version, newer-version, and older-version test matrix.
    
9. Validate failure behavior.
    
10. Perform a controlled migration with the existing Nexus backend retained until the replacement is proven.
    
11. Update deployment documentation and CI/CD publication workflows.
    
12. Only then retire the Nexus-specific implementation.
    

The migration should therefore be treated as a backend replacement rather than a redesign of the release synchronization protocol.

## Related Files

Current implementation:

```text
scripts/release/sync-to-nexus.sh
```

Systemd service:

```text
systemd/user/lesscache-sync.service
```

Systemd timer:

```text
systemd/user/lesscache-sync.timer
```

Nexus API specification:

```text
nexus-swagger.json
```

