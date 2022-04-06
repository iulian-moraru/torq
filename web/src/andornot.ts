import clone from "./clone"

enum ClauseType {
  filter,
  or,
  and
}

class Clause {
  type: ClauseType
  result?: boolean
  filter?: boolean
  childClauses: Array<Clause>
  constructor(type: ClauseType, filter?: boolean) {
    this.type = type
    this.childClauses = []
    this.filter = filter
  }
  addChildClause(clause: Clause): void {
    if (this.type === ClauseType.filter) {
      throw new Error("Child clauses must be added to AND or OR clauses")
    }
    this.childClauses.push(clause)
  }
}

const parseClause = (clause: Clause) => {
  typeSwitch: switch (clause.type) {
    case ClauseType.filter: {
      clause.result = clause.filter
      break;
    }
    case ClauseType.and: {
      for (const childClause of clause.childClauses) {
        // recursive call processing each child clause
        parseClause(childClause)
        // if any of the sibling filters are false then the AND fails, no need to process the rest
        if (childClause.result === false) {
          clause.result = false
          break typeSwitch
        }
      }
      // check that every filter is true so satisfy the AND
      if (clause.childClauses.every(sc => sc.result === true)) {
        clause.result = true
      }
      break;
    }
    case ClauseType.or: {
      for (const childClause of clause.childClauses) {
        // recursive call processing each child clause
        parseClause(childClause)
        // if any of the sibling filters are true then the OR succeeds, no need to process the rest
        if (childClause.result === true) {
          clause.result = true
          break typeSwitch
        }
      }
      // if we made it here all of the previous filters must have returned false so whole OR fails
      clause.result = false
      break;
    }
  }
}

const processQuery = (query: any): boolean => {
  // clone query to modify it and leave original untouched
  const clonedQuery = clone<Clause>(query)
  parseClause(clonedQuery)
  if (clonedQuery.result === undefined) {
    throw new Error("Query result must be true or false")
  }
  return clonedQuery.result
}


export { Clause, ClauseType, processQuery }
