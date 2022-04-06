import { Clause, ClauseType, processQuery } from "./andornot"








test('simplest query returns false', () => {
  const simplestClauseFalse = new Clause(ClauseType.filter, false)
  const result = processQuery(simplestClauseFalse)
  expect(result).toBe(false);
});

test('simplest AND query returns false', () => {
  const simpleAndClauseReturnFalse = new Clause(ClauseType.and)
  simpleAndClauseReturnFalse.addChildClause(new Clause(ClauseType.filter, false))
  simpleAndClauseReturnFalse.addChildClause(new Clause(ClauseType.filter, true))
  const result = processQuery(simpleAndClauseReturnFalse)
  expect(result).toBe(false);
});

test('simplest AND query returns true', () => {
  const simpleAndClauseReturnTrue = new Clause(ClauseType.and)
  simpleAndClauseReturnTrue.addChildClause(new Clause(ClauseType.filter, true))
  simpleAndClauseReturnTrue.addChildClause(new Clause(ClauseType.filter, true))
  const result = processQuery(simpleAndClauseReturnTrue)
  expect(result).toBe(true);
});

test('simplest OR query returns true', () => {
  const clauseOrReturnTrue = new Clause(ClauseType.or)
  clauseOrReturnTrue.addChildClause(new Clause(ClauseType.filter, true))
  clauseOrReturnTrue.addChildClause(new Clause(ClauseType.filter, false))
  const result = processQuery(clauseOrReturnTrue)
  expect(result).toBe(true);
});

test('simplest OR query returns false', () => {
  const clauseOrReturnFalse = new Clause(ClauseType.or)
  clauseOrReturnFalse.addChildClause(new Clause(ClauseType.filter, false))
  clauseOrReturnFalse.addChildClause(new Clause(ClauseType.filter, false))
  const result = processQuery(clauseOrReturnFalse);
  expect(result).toBe(false);
});

test('complex query returns true', () => {
  const simpleAndClauseReturnFalse = new Clause(ClauseType.and)
  simpleAndClauseReturnFalse.addChildClause(new Clause(ClauseType.filter, false))
  simpleAndClauseReturnFalse.addChildClause(new Clause(ClauseType.filter, true))

  const simpleAndClauseReturnTrue = new Clause(ClauseType.and)
  simpleAndClauseReturnTrue.addChildClause(new Clause(ClauseType.filter, true))
  simpleAndClauseReturnTrue.addChildClause(new Clause(ClauseType.filter, true))

  const multiClauseReturnTrue = new Clause(ClauseType.or)
  multiClauseReturnTrue.addChildClause(simpleAndClauseReturnTrue)
  multiClauseReturnTrue.addChildClause(simpleAndClauseReturnFalse)

  const result = processQuery(multiClauseReturnTrue);
  expect(result).toBe(true);
});


test('complex query returns false', () => {
  const simpleAndClauseReturnFalse = new Clause(ClauseType.and)
  simpleAndClauseReturnFalse.addChildClause(new Clause(ClauseType.filter, false))
  simpleAndClauseReturnFalse.addChildClause(new Clause(ClauseType.filter, true))

  const clauseOrReturnFalse = new Clause(ClauseType.or)
  clauseOrReturnFalse.addChildClause(new Clause(ClauseType.filter, false))
  clauseOrReturnFalse.addChildClause(new Clause(ClauseType.filter, false))

  const multiClauseReturnFalse = new Clause(ClauseType.or)
  multiClauseReturnFalse.addChildClause(clauseOrReturnFalse)
  multiClauseReturnFalse.addChildClause(simpleAndClauseReturnFalse)

  const result = processQuery(multiClauseReturnFalse);
  expect(result).toBe(false);
});
// const data = [{ a: 1, b: 2 }, { a: 2, b: 3 }, { a: 4, b: 4 }]

// const simpleQuery = {
//   $filter: false
// };

// const simpleQuery1 = {
//   $and: [
//     { $filter: true },
//     { $filter: false },
//   ]
// };

// const simpleQuery2 = {
//   $and: [
//     { $filter: true },
//     { $filter: true },
//   ]
// };

// const queryAnd = {
//   $and: [
//     { $filter: false },
//     {
//       $and: [
//         { $filter: false },
//         { $filter: false },
//         { $filter: true }]
//     }
//   ]
// };

// const queryOr = {
//   $and: [
//     { $filter: false },
//     {
//       $or: [
//         { $filter: false },
//         { $filter: false },
//         { $filter: true }]
//     }
//   ]
// };


// (() => {

//   console.log(processQuery(simpleAndClauseReturnFalse))

//   console.log(processQuery(simpleAndClauseReturnTrue))

//   console.log(processQuery(clauseOrReturnTrue))

//   console.log(processQuery(clauseOrReturnFalse))

//   console.log(processQuery(multiClauseReturnTrue))

//   console.log(processQuery(multiClauseReturnFalse))


// })()
