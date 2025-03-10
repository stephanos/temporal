// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package rule

//var (
//	varRef        = varRefType{}
//	errorRegex    = regexp.MustCompile(`Error:\s+(.*)`)
//	messagesRegex = regexp.MustCompile(`Messages:\s+(.*)`)
//)
//
//const (
//	and logicOperator = iota
//	or
//	not
//)
//
//const (
//	setOnce temporalOperator = iota
//)
//
//type (
//	errorCatcher struct {
//		lastErr ruleErr
//	}
//	RuleID           struct{}
//	logicOperator    int
//	temporalOperator int
//	requireRule      struct {
//		name     string
//		args     []any
//		varTypes []VarType
//	}
//	//ruleCond struct {
//	//	rule Rule
//	//}
//	//condRule struct {
//	//	cond  ruleCond
//	//	rules Rule
//	//}
//	logicRule struct {
//		op    logicOperator
//		rules []Rule
//	}
//	temporalRule struct {
//		op      temporalOperator
//		varType VarType
//	}
//	varRefType struct{}
//)
//
////func When(rules ...Rule) ruleCond {
////	return ruleCond{rule: logicRule{op: and, rules: rules}}
////}
////
////func (r ruleCond) Then(rules ...Rule) condRule {
////	return condRule{cond: r, rules: logicRule{op: and, rules: rules}}
////}
//
//func All(one, two Rule, rules ...Rule) Rule {
//	return logicRule{op: and, rules: append([]Rule{one, two}, rules...)}
//}
//
//func Any(one, two Rule, rules ...Rule) Rule {
//	return logicRule{op: or, rules: append([]Rule{one, two}, rules...)}
//}
//
////func (c condRule) Eval(context EvalContext) Report {
////	var res Report
////	if c.cond.rule.Eval(context).Empty() {
////		res.Merge(c.rules.Eval(context))
////	}
////	return res
////}
//
//func Either(one, two Rule) Rule {
//	return logicRule{op: or, rules: append([]Rule{one, two})}
//}
//
////func SetOnceEventually[T any]() Rule {
////	return temporalRule{op: setOnce, varType: reflect.TypeFor[T]()}
////}
//
//func (l logicRule) Eval(ctx EvalContext) Report {
//	var res Report
//	for _, rule := range l.rules {
//		// always evaluate all rules even if the operator is `and`
//		res.Merge(rule.Eval(ctx))
//	}
//
//	switch l.op {
//	case and:
//		return res
//	case or:
//		if len(l.rules) > len(res.ruleErrs) {
//			// at least one didn't fail
//			return Report{}
//		}
//	default:
//		panic(fmt.Sprintf("unsupported temporal operator: %v", l.op))
//	}
//
//	return res
//}
//
//func (t temporalRule) Eval(ctx EvalContext) Report {
//	var res Report
//	variable := ctx.Resolve(t.varType)
//	switch t.op {
//	case setOnce:
//		if len(variable.Versions) > 1 {
//			res.Add(ruleErr{
//				assertMsg: "variable was set more than once",
//				varTypes:  []VarType{t.varType},
//			})
//		}
//	default:
//		panic(fmt.Sprintf("unsupported temporal operator: %v", t.op))
//	}
//	return res
//}
//
//func (r requireRule) Eval(ctx EvalContext) Report {
//	var res Report
//	errCatcher := errorCatcher{}
//	assertion := assert.New(&errCatcher)
//
//	var varIndex int
//	var reflectArgs []reflect.Value
//	for _, arg := range r.args {
//		if arg == varRef {
//			variable := ctx.Resolve(r.varTypes[varIndex])
//			current := reflect.ValueOf(variable.CurrentOrDefault())
//			switch current.Kind() { // TODO: exhaustive?
//			case reflect.Bool:
//				current = reflect.ValueOf(current.Bool())
//			case reflect.Int, reflect.Int32, reflect.Int64:
//				current = reflect.ValueOf(current.Int())
//			case reflect.Float32, reflect.Float64:
//				current = reflect.ValueOf(current.Float())
//			case reflect.String:
//				current = reflect.ValueOf(current.String())
//			default:
//				// do nothing
//			}
//			reflectArgs = append(reflectArgs, current)
//			varIndex++
//		} else {
//			reflectArgs = append(reflectArgs, reflect.ValueOf(arg))
//		}
//	}
//
//	if !reflect.ValueOf(assertion).MethodByName(r.name).Call(reflectArgs)[0].Bool() {
//		err := errCatcher.lastErr
//		err.varTypes = r.varTypes
//		res.Add(err)
//	}
//	return res
//}
//
//func (s *errorCatcher) Errorf(format string, args ...interface{}) {
//	msg := strings.ReplaceAll(fmt.Sprintf(format, args...), "\t", " ")
//	assertMsg := errorRegex.FindStringSubmatch(msg)[1]
//	userMsg := messagesRegex.FindStringSubmatch(msg)[1]
//	if userMsg == "[]" {
//		userMsg = ""
//	}
//	s.lastErr = ruleErr{
//		assertMsg: assertMsg,
//		userMsg:   userMsg,
//	}
//}
//
//func (r ruleErr) Error() string {
//	return fmt.Sprintf("check for %v failed: '%s'", r.varTypes, r.assertMsg)
//}
//
//func (r *Report) Add(errors ...error) {
//	for _, spec := range errors {
//		r.ruleErrs = append(r.ruleErrs, spec.(ruleErr))
//	}
//}
//
//func (r *Report) Merge(report Report) {
//	r.ruleErrs = append(r.ruleErrs, report.ruleErrs...)
//}
