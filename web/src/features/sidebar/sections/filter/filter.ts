// These filter functions can operate on any data
// Rather than use generics we have opted to just use the any type
// An alternative approach could be the following generic type instead of any / unkown
// export type FilterFunc = <T extends Record<K, unknown>, K extends keyof T>(
//   input: T,
//   key: K,
//   parameter: FilterParameterType
// ) => boolean;
/* eslint-disable @typescript-eslint/no-explicit-any */
import clone from "clone";
import { SelectOption } from "features/forms/Select";

export type FilterCategoryType = "number" | "string" | "date" | "boolean" | "array" | "duration";
export type FilterParameterType = number | string | Date | boolean | Array<unknown>;
export type FilterFunc = (input: unknown, key: string, parameter: FilterParameterType) => boolean;

// available filter types that can be picked in the UI and a filter function implementation to achieve that
export const FilterFunctions = new Map<string, Map<string, FilterFunc>>([
  [
    "number",
    new Map<string, FilterFunc>([
      ["eq", (input: any, key: string, parameter: FilterParameterType) => input[key] === parameter],
      ["neq", (input: any, key: string, parameter: FilterParameterType) => input[key] !== parameter],
      ["gt", (input: any, key: string, parameter: FilterParameterType) => input[key] > parameter],
      ["gte", (input: any, key: string, parameter: FilterParameterType) => input[key] >= parameter],
      ["lt", (input: any, key: string, parameter: FilterParameterType) => input[key] < parameter],
      ["lte", (input: any, key: string, parameter: FilterParameterType) => input[key] <= parameter],
    ]),
  ],
  [
    "duration",
    new Map<string, FilterFunc>([
      ["eq", (input: any, key: string, parameter: FilterParameterType) => input[key] === parameter],
      ["neq", (input: any, key: string, parameter: FilterParameterType) => input[key] !== parameter],
      ["gt", (input: any, key: string, parameter: FilterParameterType) => input[key] > parameter],
      ["gte", (input: any, key: string, parameter: FilterParameterType) => input[key] >= parameter],
      ["lt", (input: any, key: string, parameter: FilterParameterType) => input[key] < parameter],
      ["lte", (input: any, key: string, parameter: FilterParameterType) => input[key] <= parameter],
    ]),
  ],
  [
    "string",
    new Map<string, FilterFunc>([
      [
        "like",
        (input: any, key: string, parameter: FilterParameterType) => input[key].toLowerCase().includes(parameter),
      ],
      [
        "notLike",
        (input: any, key: string, parameter: FilterParameterType) => !input[key].toLowerCase().includes(parameter),
      ],
    ]),
  ],
  [
    "boolean",
    new Map<string, FilterFunc>([
      ["eq", (input: any, key: string, parameter: FilterParameterType) => !!input[key] === parameter],
      ["neq", (input: any, key: string, parameter: FilterParameterType) => !input[key] !== parameter],
    ]),
  ],
  [
    "date",
    new Map<string, FilterFunc>([
      ["eq", (input: any, key: string, parameter: FilterParameterType) => input[key] === parameter],
      ["neq", (input: any, key: string, parameter: FilterParameterType) => input[key] !== parameter],
      ["gt", (input: any, key: string, parameter: FilterParameterType) => input[key] > parameter],
      ["gte", (input: any, key: string, parameter: FilterParameterType) => input[key] >= parameter],
      ["lt", (input: any, key: string, parameter: FilterParameterType) => input[key] < parameter],
      ["lte", (input: any, key: string, parameter: FilterParameterType) => input[key] <= parameter],
    ]),
  ],
  [
    "array",
    new Map<string, FilterFunc>([
      [
        "eq",
        (input: any, key: string, parameter: FilterParameterType) =>
          input[key].filter((value: any) => (parameter as Array<unknown>).includes(value)),
      ],
      [
        "neq",
        (input: any, key: string, parameter: FilterParameterType) =>
          !input[key].filter((value: any) => (parameter as Array<unknown>).includes(value)),
      ],
    ]),
  ],
]);

// an interface for a user configured filter with the key to operate on and value to filter by
export interface FilterInterface {
  category: FilterCategoryType;
  funcName: string;
  parameter: FilterParameterType;
  key?: string;
  selectOptions?: Array<SelectOption>;
  value?: any;
  label?: string;
}

export function applyFilters(filters: Clause, data: Array<any>): any[] {
  return data.filter((item) => processQuery(filters, item));
}

class FilterClause {
  prefix = "$filter";
  constructor(public filter: FilterInterface) {}
  get length() {
    return 1;
  }
  toJSON(): object {
    return { [this.prefix]: this.filter };
  }
}

class AndClause {
  prefix = "$and";
  childClauses: Clause[] = [];
  constructor(childClauses?: Clause[]) {
    if (childClauses) {
      this.childClauses = childClauses;
    }
  }
  get length() {
    let length = 0;
    for (const clause of this.childClauses) {
      length += clause.length;
    }
    return length;
  }
  addChildClause(clause: Clause): void {
    this.childClauses.push(clause);
  }
  toJSON(): object {
    return { [this.prefix]: this.childClauses.map((child) => child.toJSON()) };
  }
}

class OrClause extends AndClause {
  prefix = "$or";
}

type Clause = FilterClause | OrClause | AndClause;

type ClauseWithResult = Clause & {
  result?: boolean;
};

const parseClause = (clause: ClauseWithResult, data: any) => {
  typeSwitch: switch (clause.prefix) {
    case "$filter": {
      const filterClause = clause as FilterClause;
      const filterFunc = FilterFunctions.get(filterClause.filter.category)?.get(filterClause.filter.funcName);
      if (!filterFunc) {
        throw new Error("Filter function is not yet defined");
      }
      clause.result = filterFunc(data, filterClause.filter.key ?? "", filterClause.filter.parameter);
      break;
    }
    case "$and": {
      for (const childClause of (clause as AndClause).childClauses) {
        // recursive call processing each child clause
        parseClause(childClause, data);
        // if any of the sibling filters are false then the AND fails, no need to process the rest
        if ((childClause as ClauseWithResult).result === false) {
          clause.result = false;
          break typeSwitch;
        }
      }
      // check that every filter is true so satisfy the AND
      if ((clause as AndClause).childClauses.every((sc) => (sc as ClauseWithResult).result === true)) {
        clause.result = true;
      }
      break;
    }
    case "$or": {
      for (const childClause of (clause as OrClause).childClauses) {
        // recursive call processing each child clause
        parseClause(childClause, data);
        // if any of the sibling filters are true then the OR succeeds, no need to process the rest
        if ((childClause as ClauseWithResult).result === true) {
          clause.result = true;
          break typeSwitch;
        }
      }
      // if we made it here all of the previous filters must have returned false so whole OR fails
      clause.result = false;
      break;
    }
  }
};

const processQuery = (query: any, data: any): boolean => {
  // clone query to modify it and leave original untouched
  const clonedQuery = clone<ClauseWithResult>(query);
  parseClause(clonedQuery, data);
  if (clonedQuery.result === undefined) {
    throw new Error("Query result must be true or false");
  }
  return clonedQuery.result;
};

const deserialiseQuery = (query: any): Clause => {
  if (Object.keys(query)[0] === "$filter") {
    return new FilterClause(query.$filter);
  }
  if (Object.keys(query)[0] === "$and") {
    return new AndClause(query.$and.map((subclause: Clause) => deserialiseQuery(subclause)));
  }
  if (Object.keys(query)[0] === "$or") {
    return new OrClause(query.$or.map((subclause: Clause) => deserialiseQuery(subclause)));
  }
  // throw new Error("Expected JSON to contain $filter, $or or $and");
  return new AndClause();
};

const deserialiseQueryFromString = (query: string): Clause => {
  return deserialiseQuery(JSON.parse(query));
};

export { FilterClause, OrClause, AndClause, processQuery, deserialiseQuery, deserialiseQueryFromString };
export type { Clause };
