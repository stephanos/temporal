#lang racket
;; The Umpire model layer, as a Racket library and as a `#lang`.
;;
;; The surface a Model authors against: the vocabulary forms (`entity`, `enum`, `action`,
;; `observation`), the two machine forms (`machine`, `compose`), the claim forms (`property`,
;; `scenario`, `limits`, `query`, `set`), and the record form `record` for finite state structs.
;; Every form is a `syntax-parse` macro: the shape of a declaration is a grammar the macro states
;; once, so a misspelled clause, a missing one, or a step that names an undeclared action is
;; rejected at expansion with the offending token underlined, before anything runs.
;;
;; What runs when: expansion checks names and shapes (a declaration is looked up through
;; `syntax-local-value`, so the check is on the binding and not on a string); module instantiation
;; builds each machine's finite table from its step functions and, for a machine that `#:refines`
;; another, walks every row through the map and rejects the machine if a row is neither a stutter
;; nor explained by some product transition between its mapped states. Query search runs in the `test` submodule `#lang umpire` appends, or wherever
;; a `#lang racket` Model puts its own `(module+ test ...)`.
;;
;; Bodies below are sketched where the algorithm is not the point (search, the coverage cut). The
;; structs and the macro grammars are the real thing.

(require (for-syntax racket/base racket/list racket/syntax racket/struct-info syntax/parse)
         syntax/srcloc
         racket/contract
         racket/match
         racket/list
         (prefix-in rs: racket/set))   ; `set` is a DSL form here; racket/set stays reachable as rs:

(provide
 ;; vocabulary
 entity enum record action observation cases
 ;; machines
 machine compose
 ;; claims
 property scenario limits query set
 ;; runtime values a Model or a pin reads
 (struct-out step)
 machine? machine-name machine-starts machine-ends machine-timers machine-table machine-refinement
 machine-action-keys machine-stuck machine-transitions reachable-from
 table? table-states table-rows
 refinement? refinement-rows refinement-rejected
 property? scenario? limits? query? set? entity? action? observation?
 property-name scenario-name scenario-occurrences scenario-setup-state query-name set-name
 run-query query-result-outcome
 saturating-succ
 (struct-out exn:fail:umpire))

;; ---------------------------------------------------------------------------------------------
;; Runtime structs
;;
;; The DSL forms are macros named `machine`, `property`, ... so the structs of the same concept are
;; given a different struct-info name and a `make-` constructor; the accessors keep the plain
;; prefix (`machine-table`, `property-holds`) because that is what a pin reads.

;; One row of a step function's answer. `facts` is in recording order: a completion that arrives
;; before the start records the Started event first, and the list says so.
(struct step (outcome state facts) #:transparent)

(struct exn:fail:umpire exn:fail (srcloc) #:transparent)

;; A declared entity: what a machine is about, keyed by a recorded field, referring to others.
(struct entity (name key refers) #:name entity-info #:constructor-name make-entity #:transparent)

;; A declared action. `inputs` is an ordered list of (name . member-thunk); the classes of the
;; action are the cartesian product of the members, each keyed `name-in1-in2`.
(struct action (name party entity creates? inputs results schema examples)
  #:name action-info #:constructor-name make-action #:transparent)

(struct observation (name entity read)
  #:name observation-info #:constructor-name make-observation #:transparent)

;; Every (state, action class) pair with the steps it produces. States are in the record's
;; enumeration order and classes in catalog order, so the table is the same on every reading and
;; the Behavior Fingerprint can hash it.
(struct table (states classes rows) #:transparent)

;; `rows` maps a protocol row key to the product class it is a step of, or to #f for a stutter;
;; `rejected` names the first row that is neither, or #f.
(struct refinement (rows rejected) #:transparent)

(struct machine (name entity state-members starts ends timers unobservable evidence steps
                      refines map table refinement)
  #:name machine-info #:constructor-name make-machine #:transparent)

;; A same-step claim carries `when` (an action class key, or an action name for every class); a
;; transition claim carries #f and a two-argument `holds`.
(struct property (name machine when holds)
  #:name property-info #:constructor-name make-property #:transparent)

(struct scenario (name model setup-state occurrences)
  #:name scenario-info #:constructor-name make-scenario #:transparent)

(struct limits (name steps actions search)
  #:name limits-info #:constructor-name make-limits #:transparent)

;; `mode` is 'find or 'verify. A find Query is realized by a set; a verify Query is searched only.
(struct query (name mode property scenario limits)
  #:name query-info #:constructor-name make-query #:transparent)

(struct query-result (outcome witness) #:transparent)

(struct set (name purpose bind repeat queries machine cover budget)
  #:name set-info #:constructor-name make-set #:transparent)

;; ---------------------------------------------------------------------------------------------
;; Compile-time records
;;
;; Each vocabulary form binds its name to a transformer that (a) answers `syntax-local-value` with
;; a static record other forms inspect at expansion and (b) expands to the runtime value when used
;; as an expression. That is the whole trick behind the pinned errors: `machine` asks "is this
;; identifier an action?" of the binding, not of a table it might have to keep in sync.

(begin-for-syntax
  (struct static (runtime-id) #:property prop:procedure
    (λ (self stx)
      (syntax-parse stx
        [x:id (static-runtime-id self)]
        [(x:id . args) (with-syntax ([rt (static-runtime-id self)]) #'(rt . args))])))

  (struct enum-static static (constructors))          ; constructors: list of (id arity field-types)
  ;; A record's name answers three questions: as an expression it is the constructor, to
  ;; `struct-copy` and `match` it is the struct info, to `machine` it is the field list.
  (struct record-static static (fields info)           ; fields: list of (id . type-stx)
    #:property prop:struct-info (λ (self) (record-static-info self)))
  (struct entity-static static ())
  (struct action-static static (party entity inputs)) ; inputs: list of (name type-stx)
  (struct observation-static static ())
  (struct machine-static static (state-id actions timers refines)) ; refines: id or #f
  (struct property-static static (machine-id))
  (struct scenario-static static (machine-id))
  (struct limits-static static ())
  (struct query-static static ())

  (define (lookup stx pred what)
    (define v (and (identifier? stx) (syntax-local-value stx (λ () #f))))
    (unless (and v (pred v))
      (raise-syntax-error #f (format "expected ~a" what) stx))
    v)

  ;; The members of a finite type named in a field position: an enum, Bool, or (Fin n).
  (define (members-expr type-stx)
    (syntax-parse type-stx
      #:datum-literals (Bool Fin)
      [Bool #'(list #t #f)]
      [(Fin n:nat) #'(range n)]
      [e:id (lookup #'e enum-static? "an enum name")
            (with-syntax ([m (format-id #'e "~a-members" #'e)]) #'m)]
      [_ (raise-syntax-error #f "expected an enum name, Bool, or (Fin n)" type-stx)]))

  ;; A `name: value` clause, written `#:name value`, may appear once.
  (define-syntax-class party
    #:description "a party (caller, handler, worker, network, operator)"
    (pattern (~or* (~datum caller) (~datum handler) (~datum worker) (~datum network)
                   (~datum operator))))

  (define-syntax-class typed-field
    #:description "a typed field [name : Type]"
    (pattern [name:id (~datum :) type]))

  (define-syntax-class action-ref
    #:description "a declared action or a timer name"
    (pattern name:id))

  ;; `operation.schedule` is one identifier; a composition splits it at the dot.
  (define-syntax-class qualified
    #:description "member.action"
    #:attributes (member action)
    (pattern q:id
             #:do [(define parts (regexp-split #rx"[.]" (symbol->string (syntax-e #'q))))]
             #:fail-unless (= (length parts) 2) "expected member.action"
             #:with member (format-id #'q "~a" (first parts))
             #:with action (format-id #'q "~a" (second parts))))

  ;; An action with classed inputs, `[schedule unset expires unset]` or bare `backoff`. A bare
  ;; nullary constructor in an input position means its one instance.
  (define-syntax-class classed
    #:description "an action class: name, or [name input ...]"
    #:attributes (name (input 1))
    (pattern name:id #:with (input ...) #'())
    (pattern [name:id input ...])))

;; ---------------------------------------------------------------------------------------------
;; entity / enum / record / action / observation

(define-syntax (entity stx)
  (syntax-parse stx
    [(_ name:id
        (~alt (~optional (~seq #:key key:id))
              (~optional (~seq #:refer ([role:id target:id] ...)))) ...)
     #:with rt (format-id #'name "~a-entity" #'name)
     #'(begin
         (define rt (make-entity 'name '(~? key #f) '(~? ((role . target) ...) ())))
         (define-syntax name (entity-static #'rt)))]))

;; `(enum Reply syncSuccess async (handlerError [retryable : Bool]))`. A nullary constructor is a
;; symbol and a constructor with fields is a transparent struct, so `match`, `equal?` and `eq?` all
;; work and two enums of one module may both have a `scheduled`: Racket has one namespace per
;; module where Lean has one per enum, and a symbol claims no binding. `Reply-members` enumerates
;; the classes, one per constructor per assignment of its finite fields; `Reply?` is the contract.
(define-syntax (enum stx)
  (syntax-parse stx
    [(_ name:id (~and ctor (~or* c:id (c:id f:typed-field ...))) ...)
     #:with members (format-id #'name "~a-members" #'name)
     #:with pred (format-id #'name "~a?" #'name)
     #:with (nullary ...) (filter identifier? (syntax->list #'(ctor ...)))
     #:with ((sname (sfield ...) (smembers ...)) ...)
     (for/list ([c (syntax->list #'(ctor ...))] #:unless (identifier? c))
       (syntax-parse c
         [(c:id f:typed-field ...)
          #`(c (f.name ...) #,(map members-expr (syntax->list #'(f.type ...))))]))
     #:with (spred ...) (for/list ([c (syntax->list #'(sname ...))]) (format-id c "~a?" c))
     #:with ((cname arity (ftype ...)) ...)
     (for/list ([c (syntax->list #'(ctor ...))])
       (syntax-parse c
         [c:id #'(c 0 ())]
         [(c:id f:typed-field ...) #`(c #,(length (syntax->list #'(f ...))) (f.type ...))]))
     #'(begin
         (struct sname (sfield ...) #:transparent) ...
         (define (pred v) (or (memq v '(nullary ...)) (spred v) ...))
         ;; Catalog order is declaration order, then field order: `handlerError #f` before #t.
         (define members
           (append '(nullary ...)
                   (for*/list ([args (in-list (apply cartesian-product (list smembers ...)))])
                     (apply sname args))
                   ...))
         (define-syntax name
           (enum-static #'pred (list (list #'cname arity (list #'ftype ...)) ...))))]))

;; `(cases Reply reply [syncSuccess e] [(handlerError #t) e] ...)`: `match` on an enum, checked at
;; expansion for one clause per constructor (a `_` clause covers the rest). A bare constructor is
;; the pattern `'ctor`; a bare identifier that is not a constructor is refused rather than read as
;; `match` would read it, as a variable that swallows every case. This is where Racket gets the
;; exhaustiveness Lean gives `match`: not from a type, but from the enum's declaration, at the same
;; moment.
;;
;;   nexus-caller.rkt:170:5: cases: not a constructor of Reply
;;     at: syncSucess
;;   nexus-caller.rkt:168:2: cases: non-exhaustive cases on Reply: missing (operationCanceled)
(define-syntax (cases stx)
  (syntax-parse stx
    [(_ en:id subject:expr [pat body ...+] ...)
     #:do [(define info (lookup #'en enum-static? "an enum name"))
           (define ctors (enum-static-constructors info))
           (define (ctor-entry id) (assf (λ (x) (eq? (syntax-e x) (syntax-e id))) ctors))
           (define (ctor-of pat)
             (syntax-parse pat
               [(~datum _) #f]
               [c:id (unless (ctor-entry #'c)
                       (raise-syntax-error 'cases (format "not a constructor of ~a" (syntax-e #'en)) #'c))
                     #'c]
               [(c:id . _) (unless (ctor-entry #'c)
                             (raise-syntax-error 'cases (format "not a constructor of ~a" (syntax-e #'en)) #'c))
                           #'c]
               [_ (raise-syntax-error 'cases "expected a constructor pattern or _" pat)]))
           (define pats (syntax->list #'(pat ...)))
           (define named (filter-map ctor-of pats))
           (define wild? (ormap (λ (p) (syntax-parse p [(~datum _) #t] [_ #f])) pats))
           (define missing
             (filter (λ (c) (not (memf (λ (n) (eq? (syntax-e n) (syntax-e (first c)))) named))) ctors))]
     #:fail-when (and (pair? missing) (not wild?) stx)
     (format "non-exhaustive cases on ~a: missing ~a"
             (syntax-e #'en) (map (λ (c) (syntax-e (first c))) missing))
     #:with (pat* ...)
     (for/list ([p pats])
       (syntax-parse p
         [(~datum _) p]
         [c:id (if (zero? (second (ctor-entry #'c)))
                   #''c                                ; nullary: the symbol
                   (raise-syntax-error 'cases "constructor takes fields" #'c))]
         [_ p]))
     #'(match subject [pat* body ...] ...)]))

;; A finite state record: a transparent struct with `Name-members`, the cartesian product of its
;; fields in declaration order. It is what makes the table finite, and the `phase` field is what
;; `#:starts`/`#:ends` name.
(define-syntax (record stx)
  (syntax-parse stx
    [(_ name:id f:typed-field ...)
     #:with members (format-id #'name "~a-members" #'name)
     #:with (fmembers ...) (map members-expr (syntax->list #'(f.type ...)))
     #:with info (format-id #'name "~a-info" #'name)
     #:with make (format-id #'name "make-~a" #'name)
     #'(begin
         (struct name (f.name ...) #:transparent #:name info #:constructor-name make)
         (define members
           (for/list ([args (in-list (apply cartesian-product (list fmembers ...)))])
             (apply make args)))
         (define-syntax name
           (record-static #'make (list (cons #'f.name #'f.type) ...)
                          (extract-struct-info (syntax-local-value #'info)))))]))

(define-syntax (action stx)
  (syntax-parse stx
    [(_ name:id
        (~alt (~once (~seq #:party p:party) #:name "#:party clause")
              (~optional (~seq #:creates creates:id))
              (~optional (~seq #:on on:id))
              (~optional (~seq #:schema schema:str))
              (~optional (~seq #:input in:typed-field ...))
              (~optional (~seq #:results results:id))
              (~optional (~seq #:examples ([ex-class ...] (~datum ->) ex-value:str) ...))) ...)
     #:fail-when (and (attribute creates) (attribute on) #'on) "an action creates or is on an entity, not both"
     #:do [(when (attribute creates) (lookup #'creates entity-static? "a declared entity"))
           (when (attribute on) (lookup #'on entity-static? "a declared entity"))
           (when (attribute results) (lookup #'results enum-static? "an enum name"))]
     #:with rt (format-id #'name "~a-action" #'name)
     #:with (in-members ...) (map members-expr (if (attribute in) (syntax->list #'(in.type ...)) '()))
     #:with creates? (if (attribute creates) #'#t #'#f)
     #:with results-members (if (attribute results) (format-id #'results "~a-members" #'results) #'#f)
     #'(begin
         (define rt
           (make-action 'name 'p '(~? (~? creates on) #f) creates?
                        (~? (list (cons 'in.name in-members) ...) '())
                        results-members
                        (~? schema #f)
                        (list (~? (~@ (cons (list ex-class ...) ex-value) ...)))))
         (define-syntax name
           (action-static #'rt 'p '(~? (~? creates on) #f)
                          (~? (list (cons #'in.name #'in.type) ...) '()))))]))

(define-syntax (observation stx)
  (syntax-parse stx
    [(_ name:id (~alt (~once (~seq #:on on:id)) (~once (~seq #:read read:id))) ...)
     #:do [(lookup #'on entity-static? "a declared entity")]
     #:with rt (format-id #'name "~a-observation" #'name)
     #'(begin
         (define rt (make-observation 'name 'on 'read))
         (define-syntax name (observation-static #'rt)))]))

;; ---------------------------------------------------------------------------------------------
;; machine
;;
;; Two shapes: a machine from step functions, and a machine `#:from` another `#:restrict`ed to some
;; of its actions. `#:starts`/`#:ends` name phases; the runtime widens each to every state of the
;; record at that phase, which is why the protocol machine has 96 ends and not 4.
;;
;; The check a Go reader will want to see: every key under `#:steps` must be a declared action or a
;; name under `#:timers`. Anything else is reported at that identifier, at expansion:
;;
;;   nexus-caller.rkt:214:4: machine: step names an undeclared action
;;     at: handlerRepy
;;     in: (machine nexusProduct ...)

(define-syntax (machine stx)
  (syntax-parse stx
    ;; restriction
    [(_ name:id (~seq #:from from:id) (~seq #:restrict (keep:action-ref ...)))
     #:do [(define src (lookup #'from machine-static? "a declared machine"))
           (for ([k (syntax->list #'(keep ...))])
             (unless (memf (λ (a) (free-identifier=? a k))
                           (append (machine-static-actions src) (machine-static-timers src)))
               (raise-syntax-error #f "restriction names an action the machine has no step for"
                                   stx k)))]
     #:with rt (format-id #'name "~a-machine" #'name)
     #'(begin
         (define rt (restrict-machine from 'name (list (action-name keep) ...)))
         (define-syntax name
           (machine-static #'rt (machine-static-state-id (syntax-local-value #'from))
                           (list #'keep ...) '() #f)))]
    ;; from step functions
    [(_ name:id
        (~alt (~once (~seq #:for for:id) #:name "#:for clause")
              (~once (~seq #:state state:id) #:name "#:state clause")
              (~once (~seq #:starts (start-phase:id ...)) #:name "#:starts clause")
              (~once (~seq #:ends (end-phase:id ...)) #:name "#:ends clause")
              (~optional (~seq #:timers (timer:id ...)) #:defaults ([(timer 1) '()]))
              (~optional (~seq #:unobservable (silent:id ...)) #:defaults ([(silent 1) '()]))
              (~optional (~seq #:evidence ([fact:id ev:id] ...)) #:defaults ([(fact 1) '()] [(ev 1) '()]))
              (~optional (~seq #:refines refines:id))
              (~optional (~seq #:map map:expr))
              (~once (~seq #:steps ([key:action-ref fn:expr] ...)) #:name "#:steps clause")) ...)
     #:do [(lookup #'for entity-static? "a declared entity")
           (lookup #'state record-static? "a record")
           (define timers (syntax->list #'(timer ...)))
           (define (timer? k) (memf (λ (t) (free-identifier=? t k)) timers))
           (define (action? k) (action-static? (syntax-local-value k (λ () #f))))
           ;; the pinned error
           (for ([k (syntax->list #'(key ...))])
             (unless (or (timer? k) (action? k))
               (raise-syntax-error #f "step names an undeclared action" stx k)))
           (for ([s (syntax->list #'(silent ...))])
             (unless (timer? s)
               (raise-syntax-error #f "#:unobservable names a timer the machine does not own" stx s)))
           (when (attribute refines)
             (lookup #'refines machine-static? "a declared machine")
             (unless (attribute map)
               (raise-syntax-error #f "#:refines needs a #:map abstraction function" stx)))]
     #:with rt (format-id #'name "~a-machine" #'name)
     #:with (action-key ...) (filter (λ (k) (not (memf (λ (t) (free-identifier=? t k)) timers)))
                                     (syntax->list #'(key ...)))
     #:with (timer-key ...) (filter (λ (k) (memf (λ (t) (free-identifier=? t k)) timers))
                                    (syntax->list #'(key ...)))
     #:with members (format-id #'state "~a-members" #'state)
     #:with phase-of (format-id #'state "~a-phase" #'state)
     #:with refines-id (if (attribute refines) #'(quote-syntax refines) #'#f)
     #'(begin
         (define rt
           (build-machine 'name for members phase-of
                          '(start-phase ...) '(end-phase ...) '(timer ...) '(silent ...)
                          '((fact . ev) ...)
                          ;; actions carry their declaration for class enumeration; timers are
                          ;; system actions with no inputs, so they carry only a name
                          (cons (~? refines #f) (~? map #f))
                          (list (cons (action-key-decl key) fn) ...)))
         (define-syntax name
           (machine-static #'rt #'state (list #'action-key ...) (list #'timer-key ...) refines-id)))]))

;; A step key is the action's runtime declaration, or a symbol for a timer.
(define-syntax (action-key-decl stx)
  (syntax-parse stx
    [(_ k:id)
     (define v (syntax-local-value #'k (λ () #f)))
     (if (action-static? v) #'k #''k)]))

;; ---------------------------------------------------------------------------------------------
;; compose
;;
;; A product of member machines over different entities. A `#:sync` line makes two member actions
;; one action of the composition; a member action no line names stays executable on its own,
;; prefixed `member.action`. `#:starts`/`#:ends` name member phases.

(define-syntax (compose stx)
  (syntax-parse stx
    [(_ name:id
        (~alt (~once (~seq #:for (ent:id ...)))
              (~once (~seq #:state state:id))
              (~once (~seq #:members ([member:id mach:id] ...)))
              (~optional (~seq #:sync ([sync-name:id left:qualified right:qualified] ...))
                         #:defaults ([(sync-name 1) '()] [(left 1) '()] [(right 1) '()]))
              (~once (~seq #:starts (start:qualified ...)))
              (~once (~seq #:ends (end:qualified ...)))) ...)
     #:do [(for ([e (syntax->list #'(ent ...))]) (lookup e entity-static? "a declared entity"))
           (define members (syntax->list #'(member ...)))
           (define (member! m)
             (unless (memf (λ (x) (free-identifier=? x m)) members)
               (raise-syntax-error #f "not a member of the composition" stx m)))
           (for ([q (syntax->list #'(left.member ... right.member ... start.member ... end.member ...))])
             (member! q))
           (for ([m (syntax->list #'(mach ...))]) (lookup m machine-static? "a declared machine"))]
     #:with rt (format-id #'name "~a-machine" #'name)
     #'(begin
         (define rt
           (build-composition 'name (list ent ...) state
                              (list (cons 'member mach) ...)
                              (list (list 'sync-name '(left.member . left.action) '(right.member . right.action)) ...)
                              '((start.member . start.action) ...)
                              '((end.member . end.action) ...)))
         (define-syntax name (machine-static #'rt #'state (list #'sync-name ...) '() #f)))]))

;; ---------------------------------------------------------------------------------------------
;; property / scenario / limits / query / set

;; `#:when` is a classed action of the machine; with it `#:holds` takes one step, without it two.
;; The arity is checked when the module is instantiated (a lambda's arity is a runtime fact), and
;; the error carries the lambda's source location.
(define-syntax (property stx)
  (syntax-parse stx
    [(_ name:id
        (~alt (~once (~seq #:machine mach:id))
              (~optional (~seq #:when when:classed))
              (~once (~seq #:holds holds))) ...)
     #:do [(lookup #'mach machine-static? "a declared machine")]
     #:with expected-arity (if (attribute when) #'1 #'2)
     #:with rt (format-id #'name "~a-property" #'name)
     #:with when-key (if (attribute when) #'(class-key 'when.name (list when.input ...)) #'#f)
     #'(begin
         (define rt
           (make-property 'name mach when-key
                          (let ([h holds])
                            (unless (procedure-arity-includes? h expected-arity)
                              (raise (exn:fail:umpire
                                      (format "~a: #:holds takes ~a argument(s)" 'name expected-arity)
                                      (current-continuation-marks) (quote-srcloc holds))))
                            h)))
         (define-syntax name (property-static #'rt #'mach)))]))

(define-syntax (scenario stx)
  (syntax-parse stx
    [(_ name:id
        (~alt (~once (~seq #:model model:id))
              (~once (~seq #:starts start:id))
              (~once (~seq #:actions (act:classed ...)))) ...)
     #:do [(define m (lookup #'model machine-static? "a declared machine"))
           (define known (append (machine-static-actions m) (machine-static-timers m)))
           (for ([a (syntax->list #'(act.name ...))])
             ;; a composition's member actions are dotted and resolved at runtime; a plain name
             ;; must be one the machine steps on
             (unless (or (regexp-match? #rx"[.]" (symbol->string (syntax-e a)))
                         (memf (λ (k) (free-identifier=? k a)) known))
               (raise-syntax-error #f "scenario names an action the machine has no step for" stx a)))]
     #:with rt (format-id #'name "~a-scenario" #'name)
     #'(begin
         (define rt
           (make-scenario 'name model 'start
                          (list (class-key 'act.name (list act.input ...)) ...)))
         (define-syntax name (scenario-static #'rt #'model)))]))

(define-syntax (limits stx)
  (syntax-parse stx
    [(_ name:id (~alt (~once (~seq #:steps steps:nat))
                      (~once (~seq #:actions actions:nat))
                      (~once (~seq #:search search:nat))) ...)
     #:with rt (format-id #'name "~a-limits" #'name)
     #'(begin
         (define rt (make-limits 'name steps actions search))
         (define-syntax name (limits-static #'rt)))]))

;; `#:find` or `#:verify`, exactly one. The Property and the Scenario must be about the same
;; machine; that is checked on the bindings at expansion.
;; `#:named` gives the Query a runtime name other than its binding, for the one case where a Model
;; already binds the name to a constructor (nexus-caller.rkt: `handlerError` is a `Reply`).
(define-syntax (query stx)
  (syntax-parse stx
    [(_ name:id
        (~alt (~once (~or* (~seq #:find (~and prop:id (~bind [mode #''find])))
                           (~seq #:verify (~and prop:id (~bind [mode #''verify]))))
                     #:name "#:find or #:verify clause")
              (~optional (~seq #:named named:id) #:defaults ([named #'name]))
              (~once (~seq #:in scen:id))
              (~once (~seq #:limits lim:id))) ...)
     #:do [(define p (lookup #'prop property-static? "a declared property"))
           (define s (lookup #'scen scenario-static? "a declared scenario"))
           (lookup #'lim limits-static? "a declared limits")
           ;; a product Property read on the protocol machine is allowed: the refinement carries it
           (define pm (property-static-machine-id p))
           (define sm (scenario-static-machine-id s))
           (unless (or (free-identifier=? pm sm)
                       (refines? (syntax-local-value sm) pm))
             (raise-syntax-error #f "the property and the scenario are about different machines"
                                 stx #'scen))]
     #:with rt (format-id #'name "~a-query" #'name)
     #'(begin
         (define rt (make-query 'named mode prop scen lim))
         (define-syntax name (query-static #'rt)))]))

(begin-for-syntax
  ;; The scenario's machine may read a Property of the machine it refines: the refinement carries it.
  (define (refines? m target)
    (define r (machine-static-refines m))
    (and r (free-identifier=? r target))))

(define-syntax (set stx)
  (syntax-parse stx
    [(_ name:id
        (~alt (~once (~seq #:purpose (~and purpose (~or* (~datum functional) (~datum canary)
                                                        (~datum exploratory)))))
              (~once (~seq #:bind ([party:party (~and binding (~or* (~datum driven) (~datum observed)))] ...)))
              (~optional (~seq #:repeat (~and repeat (~datum implementation))))
              (~optional (~seq #:queries (q:id ...)))
              (~optional (~seq #:machine mach:id))
              (~optional (~seq #:cover (cover:id ...)))
              (~optional (~seq #:budget budget:id))) ...)
     #:fail-when (and (eq? (syntax-e #'purpose) 'exploratory) (attribute q) #'purpose)
     "an exploratory set covers a machine; it lists no queries"
     #:fail-when (and (not (eq? (syntax-e #'purpose) 'exploratory)) (not (attribute q)) #'purpose)
     "a functional or canary set lists its queries"
     #:do [(for ([x (syntax->list #'((~? (q ...) ())))]) (lookup x query-static? "a declared query"))
           (when (attribute mach) (lookup #'mach machine-static? "a declared machine"))
           (when (attribute budget) (lookup #'budget limits-static? "a declared limits"))]
     #:with rt (format-id #'name "~a-set" #'name)
     #'(begin
         (define rt
           (make-set 'name 'purpose '((party . binding) ...) '(~? repeat #f)
                     (list (~? (~@ q ...))) (~? mach #f) '(~? (cover ...) ()) (~? budget #f)))
         (define name rt))]))

;; ---------------------------------------------------------------------------------------------
;; Runtime: the finite table, refinement, search
;;
;; Contracts here are the signatures a Model's step functions must meet; a step function that
;; returns a bare step instead of a list fails at the call with the blame on the Model module.

;; Step functions take one state and zero or more inputs; only the range is fixed here, each
;; Model's own contracts fix the domain per function.
(define step-fn/c (unconstrained-domain-> (listof step?)))

;; `saturating-succ` keeps an attempt count inside its bound.
(define/contract (saturating-succ n bound)
  (-> exact-nonnegative-integer? exact-nonnegative-integer? exact-nonnegative-integer?)
  (min (add1 n) bound))

;; "handlerReply-handlerError-true", "schedule-unset-unset-unset", "backoff"
(define (class-key name inputs)
  (string-join (cons (symbol->string name) (map member->string inputs)) "-"))

(define (member->string v)
  (match v
    [#t "true"] [#f "false"]
    [(? symbol?) (symbol->string v)]
    [(? exact-integer?) (number->string v)]
    [(? struct?)
     (define-values (type _) (struct-info v))
     (define-values (nm _1 _2 _3 _4 _5 _6 _7) (struct-type-info type))
     (string-join (cons (symbol->string nm) (map member->string (rest (vector->list (struct->vector v))))) "-")]))

;; The classes of an action: one per assignment of its finite inputs, in catalog order.
(define (action-classes a)
  (match a
    [(? symbol? timer) (list (list timer '()))]
    [(action-info name _ _ _ inputs _ _ _)
     (for/list ([args (in-list (apply cartesian-product (map cdr inputs)))])
       (list name args))]))

(define/contract (build-machine name entity members phase-of starts ends timers silent evidence
                                refines+map steps)
  (-> symbol? entity? (listof any/c) procedure? (listof symbol?) (listof symbol?)
      (listof symbol?) (listof symbol?) (listof pair?)
      (cons/c (or/c machine? #f) (or/c procedure? #f))
      (listof (cons/c (or/c action? symbol?) step-fn/c))
      machine?)
  (define classes
    ;; canonical order: sorted by key, which is why the protocol catalog opens on `backoff`
    (sort (append-map (λ (s) (action-classes (car s))) steps) string<?
          #:key (λ (c) (class-key (first c) (second c)))))
  (define fn-of (for/hash ([s steps]) (values (if (symbol? (car s)) (car s) (action-name (car s))) (cdr s))))
  (define rows
    (for*/list ([st (in-list members)]
                [c (in-list classes)]
                [out (in-value (apply (hash-ref fn-of (first c)) st (second c)))]
                #:when (pair? out))
      (list st (class-key (first c) (second c)) out)))
  (define tbl (table members classes rows))
  (define (at-phase phases) (filter (λ (s) (memq (phase-of s) phases)) members))
  (define target (car refines+map))
  (define abstraction (cdr refines+map))
  (define m (make-machine name entity members (at-phase starts) (at-phase ends) timers silent
                          evidence fn-of target abstraction tbl #f))
  (define checked
    (if target (struct-copy machine-info m [refinement (check-refinement m target abstraction)]) m))
  (when (and target (refinement-rejected (machine-refinement checked)))
    ;; a rejected refinement fails module instantiation, so `raco test` and `racket` both stop here
    (raise (exn:fail:umpire (format "~a: refinement of ~a rejected at row ~a" name
                                    (machine-name target) (refinement-rejected (machine-refinement checked)))
                            (current-continuation-marks) #f)))
  checked)

;; The rule is by mapped states, not by action name: a protocol row (s, a, s') is a stutter when
;; `map s == map s'`, and a step when the product has some row from `map s` to `map s'` under any
;; class (a protocol timer row maps to the product's `timeout` row, for example). Anything else
;; rejects the machine. `rows` records which product class explained each row, or #f for a stutter.
(define (check-refinement protocol product abstraction)
  ;; (before . after) -> the first product class that moves between them, in catalog order
  (define product-moves
    (for*/fold ([h (hash)])
               ([r (in-list (table-rows (machine-table product)))]
                [s (in-list (third r))])
      (define k (cons (first r) (step-state s)))
      (if (hash-has-key? h k) h (hash-set h k (second r)))))
  (define-values (rows rejected)
    (for*/fold ([rows '()] [rejected #f])
               ([r (in-list (table-rows (machine-table protocol)))]
                [s (in-list (third r))]
                #:break rejected)
      (define before (abstraction (first r)))
      (define after (abstraction (step-state s)))
      (define key (format "~a-~a" (state-key (first r)) (second r)))
      (cond
        [(equal? before after) (values (cons (cons key #f) rows) #f)]
        [(hash-ref product-moves (cons before after) #f)
         => (λ (product-key) (values (cons (cons key product-key) rows) #f))]
        [else (values rows key)])))
  (refinement (reverse rows) rejected))

;; "scheduled-0-unset-unset-unset"
(define (state-key st)
  (string-join (map member->string (rest (vector->list (struct->vector st)))) "-"))

(define (restrict-machine from name keep)
  (define tbl (machine-table from))
  (define kept-classes (filter (λ (c) (memq (first c) keep)) (table-classes tbl)))
  (define kept-keys (map (λ (c) (class-key (first c) (second c))) kept-classes))
  (struct-copy machine-info from [name name]
               [steps (for/hash ([(k v) (machine-steps from)] #:when (memq k keep)) (values k v))]
               [table (table (table-states tbl) kept-classes
                             (filter (λ (r) (member (second r) kept-keys)) (table-rows tbl)))]))

(define (build-composition name entities state members syncs starts ends)
  ;; sketched: the product table is the synchronous product of the member tables, with each sync
  ;; pair firing as one class and every un-synced member action prefixed `member.action`
  (make-machine name entities '() starts ends '() '() '() (hash) #f #f (table '() '() '()) #f))

(define (machine-action-keys m) (map (λ (c) (class-key (first c) (second c))) (table-classes (machine-table m))))
(define (machine-transitions m) (table-rows (machine-table m)))

;; A state with no enabled step that is not an end, or #f.
(define (machine-stuck m)
  (define enabled (rs:list->set (map first (machine-transitions m))))
  (for/first ([s (machine-state-members m)]
              #:unless (or (rs:set-member? enabled s) (member s (machine-ends m))))
    s))

(define (reachable-from starts transitions)
  (let loop ([frontier starts] [seen (rs:list->set starts)] [order (reverse starts)])
    (match frontier
      ['() (reverse order)]
      [(cons s rest)
       (define next (for*/list ([r transitions] #:when (equal? (first r) s)
                                [st (third r)] #:unless (rs:set-member? seen (step-state st)))
                      (step-state st)))
       (loop (append rest (remove-duplicates next)) (rs:set-union seen (rs:list->set next))
             (append (reverse (remove-duplicates next)) order))])))

;; The search: for `find`, an exact path through the Scenario's occurrences within the Limits on
;; which the Property's `when` step satisfies `holds`; for `verify`, `holds` on every trace of the
;; path. Sketched here; the interesting part is that it reads only `machine-table`.
(define/contract (run-query q)
  (-> query? query-result?)
  (match-define (query-info _ mode prop scen lim) q)
  (define m (scenario-model scen))
  (define witness
    ;; ... enumerate traces of `scen` from its setup state through `(machine-table m)` up to
    ;; `(limits-steps lim)` steps, cut at `(limits-search lim)` candidates ...
    (search-traces m scen lim))
  (match mode
    ['find (query-result (if (ormap (λ (t) (claim-holds? prop t)) witness) 'found 'not-found) witness)]
    ['verify (query-result (if (andmap (λ (t) (claim-holds? prop t)) witness) 'verified 'refuted) witness)]))

(define (search-traces m scen lim) '())        ; elided
(define (claim-holds? prop trace) #t)          ; elided

;; ---------------------------------------------------------------------------------------------
;; `#lang umpire`
;;
;; A Model written as `#lang umpire` gets this module's forms as its language and a `test`
;; submodule appended at the end of the module, so `raco test model.rkt` builds every table,
;; checks every refinement (that already happened at instantiation) and runs every Query. The
;; module-begin collects the queries defined in the body through a compile-time parameter.

(provide (rename-out [umpire-module-begin #%module-begin])
         (except-out (all-from-out racket) #%module-begin compose set))

(define-syntax (umpire-module-begin stx)
  (syntax-parse stx
    [(_ form ...)
     #'(#%module-begin
        form ...
        (module+ test
          (require rackunit)
          (for ([q (in-list (registered-queries))])
            (define r (run-query q))
            (check-not-false (memq (query-result-outcome r) '(found verified))
                             (format "query ~a: ~a" (query-name q) (query-result-outcome r))))))]))

;; sketched: `query` would `(register-query! rt)` at instantiation
(define registry '())
(define (registered-queries) (reverse registry))

;; The reader: `#lang umpire` reads s-expressions and installs the module above as the language.
(module reader syntax/module-reader
  umpire)
