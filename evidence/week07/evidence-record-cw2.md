# CW2 evidence record

Student number: 100862618

Week and task: Week 7, CW2 Tasks 7.3–7.8

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc. GCC is configured as the compiler name, but its availability has not been confirmed.

Docker/Compose versions and image digest where relevant: Not applicable. Docker was not required for these tests.

---

Student number: 100862618

Week and task: Week 7, Tasks 7.3–7.4: Storage reads and snapshots

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc.

Docker/Compose versions and image digest where relevant: Not applicable.

Exact command and working directory:

Working directory: `C:\Users\n0c71\Desktop\100862618_6CC545_CW1\source`

```powershell
go test -tags acceptance ./internal/storage -run "TestCommitRoundTripAndIsolation|TestRejectedWritePreservesCommittedFile|TestCancellationAndAbsentFile|TestConcurrentSameNameConsistentSnapshot" -count=1
```

Expected behaviour and requirement ID: The storage implementation should correctly commit and retrieve files, preserve previously committed data when a write is rejected, handle cancellation and absent files, and provide consistent snapshots.

Observed output (attach complete, sanitised transcript):

```text
ok example.edu/6cc545/cw1/internal/storage 2.000s
```

Screenshot: `screenshots/W07_02_Storage_Reads_And_Snapshots_Passed.png`

Pass / fail / not run and explanation: PASS. The selected storage acceptance tests completed successfully.

Interpretation and limit of what the test establishes: The result confirms that the tested storage operations behaved as expected. It does not establish cross-platform behaviour or restart recovery.

Failure correction and next action: No failure occurred in this test run, so no correction was required.

AI/source acknowledgement where applicable: AI assistance was used to help organise and explain the evidence. The command and test result are from the actual project run.

---

Student number: 100862618

Week and task: Week 7, Task 7.5: Concurrent storage operations

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc.

Docker/Compose versions and image digest where relevant: Not applicable.

Exact command and working directory:

Working directory: `C:\Users\n0c71\Desktop\100862618_6CC545_CW1\source`

```powershell
go test ./internal/storage -run "TestConcurrentIndependentFiles|TestConcurrentSameNameCompleteVersions|TestCancelledWriteDoesNotHoldStoreLock" -count=1
```

Expected behaviour and requirement ID: Concurrent operations should not produce incomplete file versions or interfere with unrelated files. A cancelled write should not leave the storage lock held.

Observed output (attach complete, sanitised transcript):

```text
ok example.edu/6cc545/cw1/internal/storage 1.473s
```

Screenshot: `screenshots/W07_03_Concurrency_Tests_Passed.png`

Pass / fail / not run and explanation: PASS. All selected concurrency tests completed successfully.

Interpretation and limit of what the test establishes: The tests confirm the behaviour covered by these concurrency scenarios. They do not replace a successful Go race-detector run.

Failure correction and next action: No failure occurred in this test run, so no correction was required.

AI/source acknowledgement where applicable: AI assistance was used to help organise and explain the evidence. The command and test result are from the actual project run.

---

Student number: 100862618

Week and task: Week 7, Task 7.6: Publish/subscribe event integration

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc.

Docker/Compose versions and image digest where relevant: Not applicable.

Exact command and working directory:

Working directory: `C:\Users\n0c71\Desktop\100862618_6CC545_CW1\source`

```powershell
go test ./internal/app -run "TestWeek07_FileAddedOnlyAfterVerifiedUpload|TestWeek07_DroppedNotificationDoesNotUndoCommit" -count=1
```

Expected behaviour and requirement ID: A `file.added` event should only be published after a successful verified upload. An invalid digest should not produce a successful file-added event. A dropped notification should not undo a file that has already been committed.

Observed output (attach complete, sanitised transcript):

```text
ok example.edu/6cc545/cw1/internal/app 0.307s
```

Screenshot: `screenshots/W07_04_File_Added_Event_Integration_Tests_Passed.png`

Pass / fail / not run and explanation: PASS. Both selected application integration tests completed successfully.

Interpretation and limit of what the test establishes: The tests cover the relationship between verified storage commits and event notifications. They do not establish that every possible notification delivery failure has been handled.

Failure correction and next action: No failure occurred in this test run, so no correction was required.

AI/source acknowledgement where applicable: AI assistance was used to help organise and explain the evidence. The command and test result are from the actual project run.

---

Student number: 100862618

Week and task: Week 7, Task 7.7: Storage unit tests

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc.

Docker/Compose versions and image digest where relevant: Not applicable.

Exact command and working directory:

Working directory: `C:\Users\n0c71\Desktop\100862618_6CC545_CW1\source`

```powershell
go test ./internal/storage -count=1
```

Expected behaviour and requirement ID: All storage unit tests should complete successfully.

Observed output (attach complete, sanitised transcript):

```text
ok example.edu/6cc545/cw1/internal/storage 0.966s
```

Screenshot: `screenshots/W07_05_Storage_Unit_Tests_Passed.png`

Pass / fail / not run and explanation: PASS. The storage unit test suite completed successfully.

Interpretation and limit of what the test establishes: This confirms that the storage unit tests passed. It does not independently establish restart recovery or cross-platform atomic visibility.

Failure correction and next action: No failure occurred in this test run, so no correction was required.

AI/source acknowledgement where applicable: AI assistance was used to help organise and explain the evidence. The command and test result are from the actual project run.

---

Student number: 100862618

Week and task: Week 7, Task 7.7: Storage acceptance tests

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc.

Docker/Compose versions and image digest where relevant: Not applicable.

Exact command and working directory:

Working directory: `C:\Users\n0c71\Desktop\100862618_6CC545_CW1\source`

```powershell
go test -tags acceptance ./internal/storage -count=1 -timeout 30s
```

Expected behaviour and requirement ID: The storage acceptance test suite should complete within the specified timeout and confirm the required storage behaviour.

Observed output (attach complete, sanitised transcript):

```text
ok example.edu/6cc545/cw1/internal/storage 0.942s
```

Screenshot: `screenshots/W07_06_Storage_Acceptance_Tests_Passed.png`

Pass / fail / not run and explanation: PASS. The storage acceptance test suite completed successfully.

Interpretation and limit of what the test establishes: The result confirms that the acceptance tests passed on the current Windows/amd64 environment. It does not establish behaviour on other operating systems.

Failure correction and next action: No failure occurred in this test run, so no correction was required.

AI/source acknowledgement where applicable: AI assistance was used to help organise and explain the evidence. The command and test result are from the actual project run.

---

Student number: 100862618

Week and task: Week 7, Task 7.7: Storage race detection

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc.

Docker/Compose versions and image digest where relevant: Not applicable.

Exact command and working directory:

Working directory: `C:\Users\n0c71\Desktop\100862618_6CC545_CW1\source`

```powershell
go test -race -tags acceptance ./internal/storage -count=1 -timeout 60s
```

Expected behaviour and requirement ID: The Go race detector should run against the storage acceptance tests and report any detected data races.

Observed output (attach complete, sanitised transcript):

```text
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
```

Screenshot: `screenshots/W07_07_Storage_Race_Test_CGO_Required.png`

Pass / fail / not run and explanation: NOT RUN. The race detector could not start because CGO is disabled in the current Go environment.

Interpretation and limit of what the test establishes: This output is a toolchain limitation, not a reported data race. No conclusion about race-free behaviour can be drawn from this attempt.

Failure correction and next action: The race test could not run because `CGO_ENABLED=0`. I will enable CGO and use a compatible C compiler before running the storage race test again.

AI/source acknowledgement where applicable: AI assistance was used to help explain the CGO requirement. The error output is from the actual command run.

---

Student number: 100862618

Week and task: Week 7, Task 7.8: Full repository test suite

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc.

Docker/Compose versions and image digest where relevant: Not applicable.

Exact command and working directory:

Working directory: `C:\Users\n0c71\Desktop\100862618_6CC545_CW1\source`

```powershell
go test ./... -count=1 -timeout 60s
```

Expected behaviour and requirement ID: All packages in the repository should build and their available tests should complete successfully.

Observed output (attach complete, sanitised transcript):

```text
?       example.edu/6cc545/cw1/api/files/v1     [no test files]
?       example.edu/6cc545/cw1/cmd/ipcclient    [no test files]
?       example.edu/6cc545/cw1/cmd/node         [no test files]
?       example.edu/6cc545/cw1/cmd/request      [no test files]
ok      example.edu/6cc545/cw1/examples/firstfunction   1.841s
?       example.edu/6cc545/cw1/examples/jsonmodel       [no test files]
ok      example.edu/6cc545/cw1/internal/app     0.517s
ok      example.edu/6cc545/cw1/internal/config  1.167s
ok      example.edu/6cc545/cw1/internal/events  1.225s
ok      example.edu/6cc545/cw1/internal/httpapi 0.417s
ok      example.edu/6cc545/cw1/internal/ipc     1.200s
ok      example.edu/6cc545/cw1/internal/model  0.490s
ok      example.edu/6cc545/cw1/internal/rpcfiles 11.102s
ok      example.edu/6cc545/cw1/internal/state 0.488s
ok      example.edu/6cc545/cw1/internal/storage 0.582s
```

Screenshot: `screenshots/W07_08_Full_Repository_Tests_Passed.png`

Pass / fail / not run and explanation: PASS. All packages with tests completed successfully. Packages without test files were reported separately by Go.

Interpretation and limit of what the test establishes: This confirms that the repository test suite passed in the current environment. It does not prove that every possible runtime condition or platform has been tested.

Failure correction and next action: No failure occurred in this test run, so no correction was required.

AI/source acknowledgement where applicable: AI assistance was used to help organise and explain the evidence. The command and test output are from the actual project run.

---

Student number: 100862618

Week and task: Week 7, Task 7.8: Full repository race detection

Immutable tag and full commit hash: To be added after creating and verifying the checkpoint tag.

Date/time and timezone: 5 November 2026, 7:00 pm, Africa/Accra (UTC+00:00).

OS, architecture, go version, CGO_ENABLED, C compiler: Windows, amd64, Go 1.27.1, CGO_ENABLED=0, CC=gcc.

Docker/Compose versions and image digest where relevant: Not applicable.

Exact command and working directory:

Working directory: `C:\Users\n0c71\Desktop\100862618_6CC545_CW1\source`

```powershell
go test -race ./... -count=1 -timeout 60s
```

Expected behaviour and requirement ID: The Go race detector should run across the repository test suite and report any detected data races.

Observed output (attach complete, sanitised transcript):

```text
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
```

Screenshot: `screenshots/W07_09_Full_Repository_Race_Test_CGO_Required.png`

Pass / fail / not run and explanation: NOT RUN. The race detector could not start because CGO is disabled.

Interpretation and limit of what the test establishes: This is a CGO configuration limitation, not a reported data race. The repository has passed its normal test suite, but this attempt does not establish that the repository is race-free.

Failure correction and next action: The full repository race test could not run because `CGO_ENABLED=0`. I will enable CGO and use a compatible C compiler before rerunning the full repository race test.

AI/source acknowledgement where applicable: AI assistance was used to help explain the CGO requirement. The error output is from the actual command run.