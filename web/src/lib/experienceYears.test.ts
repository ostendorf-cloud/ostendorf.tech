import { describe, expect, it } from 'vitest'

import { experienceYearsFloor, fullYearsElapsed } from './experienceYears'

describe('fullYearsElapsed', () => {
  it('counts zero complete years the day before the first anniversary', () => {
    expect(fullYearsElapsed(new Date(2020, 5, 15), new Date(2021, 5, 14))).toBe(0)
  })

  it('counts one complete year on the anniversary itself', () => {
    expect(fullYearsElapsed(new Date(2020, 5, 15), new Date(2021, 5, 15))).toBe(1)
  })

  it('counts one complete year the day after the anniversary', () => {
    expect(fullYearsElapsed(new Date(2020, 5, 15), new Date(2021, 5, 16))).toBe(1)
  })
})

describe('experienceYearsFloor', () => {
  // The business started on 1 January 2014 (experienceYears.ts), so every
  // date within a calendar year has already completed as many full years
  // as that year's distance from 2014: the figure below only ever flips on
  // 1 January.
  it('floors 12 full years, in 2026, down to 10', () => {
    expect(experienceYearsFloor(new Date(2026, 0, 1))).toBe(10)
  })

  it('floors 12 full years, later the same year, down to 10', () => {
    expect(experienceYearsFloor(new Date(2026, 10, 13))).toBe(10)
  })

  it('floors 14 full years down to 10', () => {
    expect(experienceYearsFloor(new Date(2028, 5, 1))).toBe(10)
  })

  it('still reads 10 on the last day before the step to 15', () => {
    expect(experienceYearsFloor(new Date(2028, 11, 31))).toBe(10)
  })

  it('steps to 15 on the first day of the year that completes 15 years', () => {
    expect(experienceYearsFloor(new Date(2029, 0, 1))).toBe(15)
  })

  it('floors 19 full years down to 15', () => {
    expect(experienceYearsFloor(new Date(2033, 5, 1))).toBe(15)
  })

  it('still reads 15 on the last day before the step to 20', () => {
    expect(experienceYearsFloor(new Date(2033, 11, 31))).toBe(15)
  })

  it('steps to 20 on the first day of the year that completes 20 years', () => {
    expect(experienceYearsFloor(new Date(2034, 0, 1))).toBe(20)
  })
})
