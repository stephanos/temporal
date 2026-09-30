module umpire_tests {
  import umpire.* from "./umpire"

  run reachesBeyondOneEdgeTest = assert(
    reachableFrom(Set(0), 3, n => if (n < 2) Set(n + 1) else Set()) == Set(0, 1, 2))

  run stopsAtDeclaredBoundTest = assert(
    reachableFrom(Set(0), 1, n => Set(n + 1)) == Set(0, 1))

  run keepsStartsAtZeroBoundTest = assert(
    reachableFrom(Set(0), 0, n => Set(n + 1)) == Set(0))
}
