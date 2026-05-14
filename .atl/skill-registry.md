# Skill Registry

**Delegator use only.** Any agent that launches sub-agents reads this registry to resolve compact rules, then injects them directly into sub-agent prompts. Sub-agents do NOT read this registry or individual SKILL.md files.

See `_shared/skill-resolver.md` for the full resolution protocol.

## User Skills

| Trigger | Skill | Path |
|---------|-------|------|
| PRs over 400 lines, stacked PRs, review slices | chained-pr | /home/pau/.config/opencode/skills/chained-pr/SKILL.md |
| creating, opening, or preparing PRs for review | branch-pr | /home/pau/.config/opencode/skills/branch-pr/SKILL.md |
| writing guides, READMEs, RFCs, onboarding, architecture, or review-facing docs | cognitive-doc-design | /home/pau/.config/opencode/skills/cognitive-doc-design/SKILL.md |
| PR feedback, issue replies, reviews, Slack messages, or GitHub comments | comment-writer | /home/pau/.config/opencode/skills/comment-writer/SKILL.md |
| creating GitHub issues, bug reports, or feature requests | issue-creation | /home/pau/.config/opencode/skills/issue-creation/SKILL.md |
| judgment day, dual review, adversarial review, juzgar | judgment-day | /home/pau/.config/opencode/skills/judgment-day/SKILL.md |
| Go tests, go test coverage, Bubbletea teatest, golden files | go-testing | /home/pau/.config/opencode/skills/go-testing/SKILL.md |
| new skills, agent instructions, documenting AI usage patterns | skill-creator | /home/pau/.config/opencode/skills/skill-creator/SKILL.md |
| implementation, commit splitting, chained PRs, or keeping tests and docs with code | work-unit-commits | /home/pau/.config/opencode/skills/work-unit-commits/SKILL.md |

## Compact Rules

### chained-pr
- Split PRs >400 lines into chained, reviewable units
- Each PR must be independently reviewable and testable
- Preserve test+docs with code in same PR
- Use git worktrees or feature branches for parallel work
- Document dependencies between chained PRs clearly

### branch-pr
- Create issues before PRs — define the problem first
- Run issue-first checks: problem clear, acceptance criteria defined
- Use conventional commits only, no "Co-Authored-By" AI attribution
- Keep PRs focused on single concern
- Include test changes with implementation

### cognitive-doc-design
- Design docs that reduce cognitive load, not increase it
- Structure: Goal → Context → Decision → Consequences
- Use visual hierarchy: headings, lists, code blocks
- Avoid walls of text — break into scannable sections
- Include examples for complex concepts

### comment-writer
- Write warm, direct collaboration comments
- Start with validation, then explain issues technically
- Use "we" language, not accusatory "you"
- Provide examples when suggesting changes
- Balance critique with appreciation for good work

### issue-creation
- Create issues with clear problem statements
- Include acceptance criteria in every issue
- Add context: what, why, impact
- Link to related issues/PRs
- Use labels and milestones appropriately

### judgment-day
- Run blind dual review before merging significant changes
- Reviewer A and Reviewer B review independently
- Fix all confirmed issues before re-judging
- Document decisions and tradeoffs
- Use adversarial thinking to find edge cases

### go-testing
- Use Go standard testing (`go test`) with table-driven tests
- Achieve coverage with `go test -cover`
- Use `teatest` for Bubbletea component testing
- Maintain golden files for output-sensitive tests
- Keep test files alongside source (`*_test.go`)

### skill-creator
- Create LLM-first skills with valid frontmatter
- Structure: Activation Contract → Hard Rules → Decision Gates → Execution Steps → Output Contract
- Keep skills under 450 tokens; move examples to `references/`
- Use trigger-first descriptions (max 250 chars)
- Include observable rules and clear output contracts

### work-unit-commits
- Plan commits as reviewable work units
- Each commit should be independently testable
- Keep tests and docs with the code they verify
- Use conventional commits for clear history
- Split large changes into logical units

## Project Conventions

| File | Path | Notes |
|------|------|-------|
| AGENTS.md | /home/pau/.config/opencode/AGENTS.md | Index — defines persona, rules, language, behavior |

Read the convention files listed above for project-specific patterns and rules. All referenced paths have been extracted — no need to read index files to discover more.
