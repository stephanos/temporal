package helper.references

import framework.dsl.*

object Helpers:
  def twoHop[S <: Product](
      c: Composition[S],
      member: S => Any,
      action: Class | Action[?]
  ): Composed =
    own(c, member, action)
  def curried[S <: Product](
      c: Composition[S]
  )(member: S => Any, action: Class | Action[?]): Composed =
    c.own(member, action)
  def own[S <: Product](c: Composition[S], member: S => Any, action: Class | Action[?]): Composed =
    c.own(member, action)
  def synced[S <: Product](c: Composition[S], move: S => (Any, Class | Action[?])): Composed =
    c.synced(move)
  def forward(value: Composed): Composed = value
  def member[S <: Product](c: Composition[S], move: S => (Any, Model)): Composition[S] =
    c.withMember(move)
  def unrelated[S, M](read: S => M): S => M = read
