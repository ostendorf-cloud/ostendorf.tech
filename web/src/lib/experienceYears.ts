/**
 * The business started on 1 January 2014. The exact day matters, not just
 * the year: because the start date sits on the first of January, every
 * date within a given calendar year has already completed as many full
 * years as that year's distance from 2014, so the figure below ticks over
 * on 1 January of the relevant year rather than partway through it.
 */
const BUSINESS_FOUNDED = new Date(2014, 0, 1)

/**
 * fullYearsElapsed is the number of complete years between start and now,
 * counting a year as complete only once its calendar anniversary (month
 * and day) has passed, not merely once the calendar year has ticked over.
 * A pure function of its two arguments, so a caller fixes "now" instead of
 * this reaching for the current date itself.
 */
export function fullYearsElapsed(start: Date, now: Date): number {
  let years = now.getFullYear() - start.getFullYear()
  const anniversaryPassed =
    now.getMonth() > start.getMonth() ||
    (now.getMonth() === start.getMonth() && now.getDate() >= start.getDate())
  if (!anniversaryPassed) {
    years -= 1
  }
  return years
}

/**
 * experienceYearsFloor is the figure the "Über mich" facts render: the
 * full years the business has run since BUSINESS_FOUNDED, floored to the
 * nearest multiple of five so a visitor reads a round number. The counter
 * component appends the trailing "+" itself; this returns the bare number.
 */
export function experienceYearsFloor(now: Date): number {
  return Math.floor(fullYearsElapsed(BUSINESS_FOUNDED, now) / 5) * 5
}
