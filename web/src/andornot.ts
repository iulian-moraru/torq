const data = [{ a: 1, b: 2 }, { a: 2, b: 3 }, { a: 4, b: 4 }]

const simpleQuery = {
  $filter: false
};

const simpleQuery1 = {
  $and: [
    { $filter: true },
    { $filter: false },
  ]
};

const simpleQuery2 = {
  $and: [
    { $filter: true },
    { $filter: true },
  ]
};

const queryAnd = {
  $and: [
    { $filter: false },
    {
      $and: [
        { $filter: false },
        { $filter: false },
        { $filter: true }]
    }
  ]
};

const queryOr = {
  $and: [
    { $filter: false },
    {
      $or: [
        { $filter: false },
        { $filter: false },
        { $filter: true }]
    }
  ]
};


const recursive = (clause: any) => {
  if (Object.keys(clause)[0] === "$filter") {
    // process filter
    clause.$result = clause.$filter
    return
  }
  if (Object.keys(clause)[0] === "$and") {
    for (const subClause of clause.$and) {
      recursive(subClause)
      // short circuit if any are false

    }
  }
}

enum ClauseType {
  filter,
  or,
  and
}

class Clause {
  parent?: Clause
  type: ClauseType
  result?: boolean
  filter?: boolean
  ChildClauses: Array<Clause>
  constructor(type: ClauseType, filter?: boolean) {
    this.type = type
    this.ChildClauses = []
    this.filter = filter
  }
  addChildClause(clause: Clause): void {
    if (this.type === ClauseType.filter) {
      throw new Error("Child clauses must be added to AND or OR clauses")
    }
    clause.parent = this
    this.ChildClauses.push(clause)
  }
}

const simplestClauseFalse = new Clause(ClauseType.filter, false)

const simpleAndClause = new Clause(ClauseType.and)
simpleAndClause.addChildClause(new Clause(ClauseType.filter, false))
simpleAndClause.addChildClause(new Clause(ClauseType.filter, true))

const simpleAndClauseReturnTrue = new Clause(ClauseType.and)
simpleAndClause.addChildClause(new Clause(ClauseType.filter, true))
simpleAndClause.addChildClause(new Clause(ClauseType.filter, true))

const collapse = (clause: Clause): boolean | undefined => {
  if (clause.type === ClauseType.filter) {
    // process filter
    clause.result = clause.filter
  }
  if (clause.type === ClauseType.and) {
    for (const childClause of clause.ChildClauses) {
      collapse(childClause)
      // if any of the sibling filters are false then the AND fails, no need to process the rest
      if (childClause.result === false) {
        clause.result = false
        break;
      }
    }
    if (clause.ChildClauses.every(sc => sc.result === true)) {
      clause.result = true
    }
  }
  //we are the top level so return the answer
  if (!clause.parent) {
    if (clause.result === undefined) {
      throw new Error("Filter result must be true or false")
    }
    return clause.result
  }

  // const andWork = [];
  // if (Object.keys(clause)[0] === "$and") {
  //   andWork.push(...clause.$and)

  //   remainingWork.push(andWork.shift())

  //   while (remainingWork.length) {
  //     const subClause = remainingWork.shift()
  //     console.log(subClause)
  //     if (Object.keys(subClause)[0] === "$filter") {
  //       // process filter
  //       subClause.$result = subClause.$filter
  //       // if any filter is false, whole AND is false
  //       if (!subClause["$filter"]) {
  //         clause["$result"] = false;
  //         continue
  //       }
  //       remainingWork.push(andWork.shift())
  //     }
  //   }
  // }
}


(() => {

  const remainingWork: Clause[] = new Array<Clause>();
  // remainingWork.push(simplestClauseFalse)
  // collapse(remainingWork)
  // console.log(collapse(simplestClauseFalse))
  // console.log(simplestClauseFalse)

  console.log(collapse(simpleAndClause))
  console.log(simpleAndClause)

  console.log(collapse(simpleAndClauseReturnTrue))
  console.log(simpleAndClauseReturnTrue)
  // recursive(simpleQuery2)
  // console.log(JSON.stringify(simpleQuery2))

  // recursive(queryAnd)
  // console.log(JSON.stringify(queryAnd))

  // recursive(queryOr)
  // console.log(JSON.stringify(queryAnd))
})()

export { }
