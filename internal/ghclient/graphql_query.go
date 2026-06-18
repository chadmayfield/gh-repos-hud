package ghclient

// repoFields is the per-repository selection set shared by the org and user
// queries. It fetches, in one round trip: default branch + short SHA + CI
// rollup, latest release/tags, open Dependabot alerts (with severity), and
// open PRs (enough to classify bot-vs-human, mergeable, and CI-green).
const repoFields = `
fragment repoFields on Repository {
  name
  url
  isArchived
  isPrivate
  isFork
  hasVulnerabilityAlertsEnabled
  defaultBranchRef {
    name
    target {
      ... on Commit {
        oid
        abbreviatedOid
        statusCheckRollup { state }
      }
    }
  }
  latestRelease { tagName }
  refs(refPrefix: "refs/tags/", first: 100, orderBy: {field: TAG_COMMIT_DATE, direction: DESC}) {
    nodes {
      name
      target {
        __typename
        ... on Tag { target { oid } }
        ... on Commit { oid }
      }
    }
  }
  vulnerabilityAlerts(states: OPEN, first: 50) {
    totalCount
    nodes { securityVulnerability { severity } }
  }
  pullRequests(states: OPEN, first: 20, orderBy: {field: UPDATED_AT, direction: DESC}) {
    totalCount
    nodes {
      isDraft
      mergeable
      author { login __typename }
      commits(last: 1) { nodes { commit { statusCheckRollup { state } } } }
    }
  }
}`

const orgReposQuery = repoFields + `
query OrgRepos($login: String!, $cursor: String) {
  rateLimit { remaining limit resetAt cost }
  organization(login: $login) {
    repositories(first: 100, after: $cursor, ownerAffiliations: OWNER, orderBy: {field: PUSHED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes { ...repoFields }
    }
  }
}`

const userReposQuery = repoFields + `
query UserRepos($login: String!, $cursor: String) {
  rateLimit { remaining limit resetAt cost }
  user(login: $login) {
    repositories(first: 100, after: $cursor, ownerAffiliations: OWNER, orderBy: {field: PUSHED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes { ...repoFields }
    }
  }
}`
