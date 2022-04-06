import { AndClause, OrClause, FilterClause, processQuery } from "./andornot"

test('simplest query returns false', () => {
  const simplestClauseFalse = new FilterClause(false)
  const result = processQuery(simplestClauseFalse)
  expect(result).toBe(false);
});

test('simplest AND query returns false', () => {
  const simpleAndClauseReturnFalse = new AndClause()
  simpleAndClauseReturnFalse.addChildClause(new FilterClause(false))
  simpleAndClauseReturnFalse.addChildClause(new FilterClause(true))
  const result = processQuery(simpleAndClauseReturnFalse)
  expect(result).toBe(false);
});

test('simplest AND query returns true', () => {
  const simpleAndClauseReturnTrue = new AndClause()
  simpleAndClauseReturnTrue.addChildClause(new FilterClause(true))
  simpleAndClauseReturnTrue.addChildClause(new FilterClause(true))
  const result = processQuery(simpleAndClauseReturnTrue)
  expect(result).toBe(true);
});

test('simplest OR query returns true', () => {
  const clauseOrReturnTrue = new OrClause()
  clauseOrReturnTrue.addChildClause(new FilterClause(true))
  clauseOrReturnTrue.addChildClause(new FilterClause(false))
  const result = processQuery(clauseOrReturnTrue)
  expect(result).toBe(true);
});

test('simplest OR query returns false', () => {
  const clauseOrReturnFalse = new OrClause()
  clauseOrReturnFalse.addChildClause(new FilterClause(false))
  clauseOrReturnFalse.addChildClause(new FilterClause(false))
  const result = processQuery(clauseOrReturnFalse);
  expect(result).toBe(false);
});

test('complex query returns true', () => {
  const simpleAndClauseReturnFalse = new AndClause()
  simpleAndClauseReturnFalse.addChildClause(new FilterClause(false))
  simpleAndClauseReturnFalse.addChildClause(new FilterClause(true))

  const simpleAndClauseReturnTrue = new AndClause()
  simpleAndClauseReturnTrue.addChildClause(new FilterClause(true))
  simpleAndClauseReturnTrue.addChildClause(new FilterClause(true))

  const multiClauseReturnTrue = new OrClause()
  multiClauseReturnTrue.addChildClause(simpleAndClauseReturnTrue)
  multiClauseReturnTrue.addChildClause(simpleAndClauseReturnFalse)

  const result = processQuery(multiClauseReturnTrue);
  expect(result).toBe(true);
});

test('complex query returns false', () => {
  const simpleAndClauseReturnFalse = new AndClause()
  simpleAndClauseReturnFalse.addChildClause(new FilterClause(false))
  simpleAndClauseReturnFalse.addChildClause(new FilterClause(true))

  const clauseOrReturnFalse = new OrClause()
  clauseOrReturnFalse.addChildClause(new FilterClause(false))
  clauseOrReturnFalse.addChildClause(new FilterClause(false))

  const multiClauseReturnFalse = new OrClause()
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
