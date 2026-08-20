import Temporal.Product.NexusLifecycle

namespace Umpire3.Temporal.Product.NexusLifecycle.Tests

example : edges.length = 17 := edge_count

example : next .unspecified .schedule = [.scheduled] := by rfl

example : next .scheduled .succeed = [.succeeded] := by rfl

example : next .started .succeed = [.succeeded] := by rfl

example : next .backingOff .timeout = [.timedOut] := by rfl

example : next .unspecified .reject = [.rejected] := by rfl

example : next .backingOff .succeed = [] := by rfl

end Umpire3.Temporal.Product.NexusLifecycle.Tests
