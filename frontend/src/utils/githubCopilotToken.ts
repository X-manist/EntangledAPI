export function isGitHubPersonalAccessToken(token: string): boolean {
  const value = token.trim()
  return value.startsWith('ghp_') || value.startsWith('github_pat_')
}

export const githubCopilotOAuthTokenHint =
  'Use a GitHub OAuth token from Copilot sign-in (usually ghu_/gho_). GitHub PATs (ghp_/github_pat_) are not accepted by the Copilot token exchange endpoint.'

export const githubCopilotPATError =
  'GitHub Copilot does not accept GitHub PATs (ghp_/github_pat_) here. Use a GitHub OAuth token from Copilot sign-in (ghu_/gho_) or choose API Key and paste a Copilot session token.'
