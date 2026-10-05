/* The checked-in IR file of the Nexus caller close and reset designs (umpire.irFile). */
package temporal
package features.nexuscaller
package closepolicy

import umpire.*

// Each design's Queries are a root, and so is each progress claim.
val nexusCloseFile = irFile("nexus-close")(
  rejectAfterCloseQueries,
  ackByOriginalQueries,
  retainAndRouteQueries,
  forgetsCancelOnResetQueries,
  truncatesOnResetQueries,
  retainAndRouteBoundedRetryQueries,
  rejectAfterCloseWithDeadlineQueries,
  ackByOriginalWithDeadlineQueries,
  retainAndRouteWithDeadlineQueries,
  rejectAfterCloseProgress,
  ackByOriginalProgress,
  retainAndRouteProgress,
  retainAndRouteBoundedRetryProgress,
  rejectAfterCloseWithDeadlineProgress,
  ackByOriginalWithDeadlineProgress,
  retainAndRouteWithDeadlineProgress,
  retainedReachesOwner,
  retainedWaitsWithoutRecovery,
  retainedReachesOwnerBoundedRetry
)
