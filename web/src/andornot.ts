import clone from "./clone"

class FilterClause {
  prefix: string = "$filter"
  constructor(public filter: boolean) { }
}

class AndClause {
  prefix: string = "$and"
  childClauses: Clause[] = []
  constructor(childClauses?: Clause[]) {
    if (childClauses) {
      this.childClauses = childClauses
    }
  }
  addChildClause(clause: Clause): void {
    this.childClauses.push(clause)
  }
}

class OrClause extends AndClause {
  prefix: string = "$or"
}

type Clause = FilterClause | OrClause | AndClause

type ClauseWithResult = Clause & {
  result?: boolean
}

const parseClause = (clause: ClauseWithResult) => {
  typeSwitch: switch (clause.prefix) {
    case "$filter": {
      clause.result = (clause as FilterClause).filter
      break;
    }
    case "$and": {
      for (const childClause of (clause as AndClause).childClauses) {
        // recursive call processing each child clause
        parseClause(childClause)
        // if any of the sibling filters are false then the AND fails, no need to process the rest
        if ((childClause as ClauseWithResult).result === false) {
          clause.result = false
          break typeSwitch
        }
      }
      // check that every filter is true so satisfy the AND
      if ((clause as AndClause).childClauses.every(sc => (sc as ClauseWithResult).result === true)) {
        clause.result = true
      }
      break;
    }
    case "$or": {
      for (const childClause of (clause as OrClause).childClauses) {
        // recursive call processing each child clause
        parseClause(childClause)
        // if any of the sibling filters are true then the OR succeeds, no need to process the rest
        if ((childClause as ClauseWithResult).result === true) {
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
  const clonedQuery = clone<ClauseWithResult>(query)
  parseClause(clonedQuery)
  if (clonedQuery.result === undefined) {
    throw new Error("Query result must be true or false")
  }
  return clonedQuery.result
}


export { FilterClause, OrClause, AndClause, processQuery }
