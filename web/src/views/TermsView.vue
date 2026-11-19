<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { GovueLegalPage } from '@ostendorf-cloud/govue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const germanHref = computed(() => router.resolve({ name: 'terms', params: { locale: 'de' } }).href)
</script>

<template>
  <!-- The site's own skip link and language switch are in App.vue and the header. -->
  <GovueLegalPage
    page="terms"
    :title="t('legal.agb.title')"
    :skip-link="false"
    :language-switch="false"
  >
    <div v-if="route.params.locale === 'de'" class="flex flex-col gap-4 text-text">
      <h2 class="text-xl font-semibold">§ 1 Geltungsbereich</h2>
      <p>
        Diese Allgemeinen Geschäftsbedingungen gelten für alle Verträge zwischen Simon Ostendorf
        (nachfolgend „Auftragnehmer“) und dem jeweiligen Auftraggeber (nachfolgend „Veranstalter“)
        über die Erbringung von tontechnischen Dienstleistungen sowie die Bereitstellung von
        Equipment. Abweichende Bedingungen des Veranstalters werden nur anerkannt, wenn der
        Auftragnehmer diesen ausdrücklich schriftlich zugestimmt hat.
      </p>

      <h2 class="text-xl font-semibold">§ 2 Leistungsumfang</h2>
      <p>Der Leistungsumfang ergibt sich aus dem jeweiligen schriftlichen Angebot.</p>
      <p>
        Ausdrücklich nicht Bestandteil der Leistung sind Tätigkeiten als Sicherheitsbeauftragter für
        Veranstaltungstechnik (SiMA), Rigging, Traversenbau, Bühnenbau, Elektroinstallation sowie
        alle weiteren gewerke- oder erlaubnispflichtigen Tätigkeiten nach DGUV und MVStättVO. Der
        Auftragnehmer ist keine zugelassene Fachkraft für diese Bereiche. Die Verantwortung und
        Haftung für diese Bereiche liegt ausschließlich beim Veranstalter.
      </p>

      <h2 class="text-xl font-semibold">§ 3 Angebot und Vertragsschluss</h2>
      <p>
        Angebote des Auftragnehmers sind freibleibend und für 14 Tage ab Ausstellungsdatum gültig,
        sofern keine andere Frist angegeben ist. Der Vertrag kommt durch schriftliche
        Auftragsbestätigung (E-Mail genügt) oder durch Beginn der Leistungserbringung zustande.
        Mündliche Nebenabreden bedürfen der schriftlichen Bestätigung.
      </p>

      <h2 class="text-xl font-semibold">§ 4 Vergütung und Zahlungsbedingungen</h2>
      <p>
        Die vereinbarte Vergütung ist je Veranstaltungstag fällig. Rechnungen sind innerhalb von 14
        Tagen nach Rechnungsstellung ohne Abzug zahlbar. Bei Konzertreihen oder Touren mit mehreren
        Terminen behält sich der Auftragnehmer eine Abschlagszahlung von 30 % vor Tourstart vor.
      </p>
      <p>Gemäß § 19 UStG wird aufgrund der Kleinunternehmerregelung keine Umsatzsteuer erhoben.</p>
      <p>
        Bei Zahlungsverzug ist der Auftragnehmer berechtigt, Verzugszinsen in Höhe von 5 % p.a. über
        dem jeweils geltenden Basiszinssatz zu berechnen.
      </p>

      <h2 class="text-xl font-semibold">§ 5 Stornierung und Ausfallhonorar</h2>
      <p>
        Bei Absage oder Verschiebung eines oder mehrerer Termine durch den Veranstalter gelten
        folgende Stornierungsgebühren bezogen auf das vereinbarte Honorar je abgesagtem Termin:
      </p>
      <ul class="list-disc pl-6">
        <li>Absage mehr als 60 Tage vor dem Termin: 25 % des Honorars</li>
        <li>Absage 30–60 Tage vor dem Termin: 50 % des Honorars</li>
        <li>Absage 14–29 Tage vor dem Termin: 75 % des Honorars</li>
        <li>Absage weniger als 14 Tage vor dem Termin: 100 % des Honorars</li>
      </ul>
      <p>
        Bereits entstandene Kosten (z. B. Equipment-Anschaffungen oder -Mieten, die nachweislich für
        die gebuchte Produktion getätigt wurden) sind unabhängig vom Zeitpunkt der Absage
        vollständig zu erstatten. Der Auftragnehmer ist verpflichtet, anderweitig erzielte Einnahmen
        für den abgesagten Zeitraum anzurechnen.
      </p>

      <h2 class="text-xl font-semibold">§ 6 Arbeitsbedingungen und Arbeitszeit</h2>
      <p>
        Ein Veranstaltungstag (Tagessatz) umfasst bis zu 11 Stunden Arbeitszeit inklusive einer
        Stunde Pause. Wird diese Arbeitszeit überschritten, wird ein zusätzlicher Tagessatz in
        gleicher Höhe berechnet, sofern keine abweichende Vereinbarung getroffen wurde.
      </p>
      <p>
        Der Auftragnehmer ist berechtigt, seine Tätigkeit zu unterbrechen oder einzustellen, wenn
        die Arbeitsbedingungen vor Ort seine Sicherheit oder die Sicherheit des Equipments gefährden
        und der Veranstalter trotz Aufforderung keine Abhilfe schafft. In diesem Fall besteht kein
        Anspruch des Veranstalters auf Rückerstattung bereits geleisteter Zahlungen.
      </p>

      <h2 class="text-xl font-semibold">§ 7 Equipment – Bereitstellung und Haftung</h2>
      <p>
        Das vom Auftragnehmer mitgebrachte Equipment wird in einwandfreiem, funktionstüchtigem
        Zustand zur Verfügung gestellt. Der Veranstalter haftet für Beschädigungen oder Verlust des
        Equipments ab dem Zeitpunkt der Anlieferung bis zum Abtransport. Dies gilt insbesondere für
        Schäden durch unsachgemäße Handhabung durch Dritte (Publikum, lokales Personal), Diebstahl
        sowie Elementarschäden.
      </p>
      <p>
        Beschädigte oder verlorene Geräte werden zum aktuellen Wiederbeschaffungspreis (Neupreis) in
        Rechnung gestellt. Eventuelle Schäden sind unverzüglich, spätestens bei Abbau zu melden. Der
        Auftragnehmer empfiehlt dem Veranstalter den Abschluss einer
        Veranstalterhaftpflichtversicherung.
      </p>
      <p>
        Der Auftragnehmer haftet nicht für Schäden, die durch mangelhafte Stromversorgung
        (Schwankungen, Überspannung) seitens des Veranstalters oder der Venue entstehen.
      </p>

      <h2 class="text-xl font-semibold">§ 8 Pflichten des Veranstalters</h2>
      <p>Der Veranstalter stellt folgende Voraussetzungen kostenfrei zur Verfügung:</p>
      <ul class="list-disc pl-6">
        <li>Freier und rechtzeitiger Zugang zur Venue ab dem vereinbarten Get-in-Zeitpunkt</li>
        <li>
          Ausreichend dimensionierter Bühnenstrom (230V/16A Schuko, getrennt von Lichtstrom) sowie
          FOH-Strom
        </li>
        <li>FOH-Position im Zuschauerraum (nicht in Regiekabine), mind. 1 m Breite</li>
        <li>Netzwerkverbindung (Cat 5e oder höher) zwischen FOH und Bühne, sofern vereinbart</li>
        <li>
          Lokales technisches Personal (Licht- und Haustontechniker, Stagehands) gemäß
          Bühnenanweisung
        </li>
        <li>
          Einholung sämtlicher behördlicher Genehmigungen, insbesondere Schallschutzgenehmigungen
          und Ausnahmegenehmigungen für Lärmpegel
        </li>
        <li>Beauftragung und Kosten für etwaige Schallpegelmessungen oder Lärmschutzgutachten</li>
        <li>Einhaltung und Überwachung der behördlich zugelassenen Emissionsgrenzwerte</li>
      </ul>
      <p>
        Der Auftragnehmer wird auf Anweisung des Veranstalters auf Pegelgrenzen Rücksicht nehmen,
        trägt jedoch keine Verantwortung und keine Haftung für behördliche Maßnahmen, die aus der
        Überschreitung genehmigter Grenzwerte resultieren.
      </p>
      <p>
        Eventuelle GEMA-Gebühren und behördliche Genehmigungen trägt der Veranstalter. Bei
        Konzertreihen sind die Venue-Specs spätestens 4 Wochen vor der jeweiligen Veranstaltung an
        den Auftragnehmer zu übermitteln.
      </p>

      <h2 class="text-xl font-semibold">§ 9 Höhere Gewalt</h2>
      <p>
        Kann der Auftragnehmer seine Leistung aufgrund höherer Gewalt (Naturkatastrophen,
        behördliche Anordnungen, Stromausfall, Pandemie o. ä.) nicht erbringen, ist er von der
        Leistungspflicht befreit. Bereits geleistete Zahlungen für den betroffenen Zeitraum werden
        anteilig erstattet. Weitergehende Schadensersatzansprüche sind in diesem Fall
        ausgeschlossen.
      </p>

      <h2 class="text-xl font-semibold">§ 10 Referenzen und Bildrechte</h2>
      <p>
        Der Auftragnehmer ist berechtigt, die erbrachte Leistung zu Referenz- und Werbezwecken zu
        erwähnen. Sofern der Veranstalter nicht schriftlich widerspricht, darf der Auftragnehmer
        eigene Aufnahmen vom Arbeitsumfeld (nicht vom Künstler oder der Show selbst) zu
        Portfoliozwecken verwenden.
      </p>

      <h2 class="text-xl font-semibold">§ 11 Datenschutz</h2>
      <p>
        Im Zusammenhang mit der Anbahnung und Durchführung von Verträgen erhebt und speichert der
        Auftragnehmer personenbezogene Daten des Veranstalters ausschließlich zum Zweck der
        Vertragserfüllung. Eine Weitergabe an Dritte erfolgt nicht, es sei denn, dies ist gesetzlich
        vorgeschrieben. Auf Wunsch werden die Daten nach Abschluss der Geschäftsbeziehung gelöscht,
        soweit keine gesetzlichen Aufbewahrungsfristen entgegenstehen.
      </p>

      <h2 class="text-xl font-semibold">§ 12 Schlussbestimmungen</h2>
      <p>
        Es gilt das Recht der Bundesrepublik Deutschland. Gerichtsstand für alle Streitigkeiten ist
        Aachen, soweit gesetzlich zulässig. Sollten einzelne Bestimmungen dieser AGB unwirksam sein
        oder werden, bleibt die Wirksamkeit der übrigen Bestimmungen unberührt. An Stelle der
        unwirksamen Bestimmung gilt die gesetzlich zulässige Regelung, die dem ursprünglichen Zweck
        am nächsten kommt.
      </p>
      <p>
        Mündliche Nebenabreden bestehen nicht. Änderungen und Ergänzungen bedürfen der Schriftform.
      </p>
    </div>

    <div v-else class="flex flex-col gap-4 text-text">
      <p class="text-text-muted">
        {{ t('legal.bindingNotice') }}
        <a class="text-accent-text hover:underline" :href="germanHref">{{
          t('legal.viewGerman')
        }}</a>
      </p>

      <h2 class="text-xl font-semibold">Section 1 Scope of application</h2>
      <p>
        These General Terms and Conditions apply to all contracts between Simon Ostendorf
        (hereinafter the "Contractor") and the respective client (hereinafter the "Organiser") for
        the provision of audio engineering services and the provision of equipment. Deviating terms
        proposed by the Organiser are only recognised where the Contractor has expressly agreed to
        them in writing.
      </p>

      <h2 class="text-xl font-semibold">Section 2 Scope of services</h2>
      <p>The scope of services results from the respective written quotation.</p>
      <p>
        Expressly not part of the service are activities as a safety officer for event technology
        ("SiMA"), rigging, truss construction, stage construction, electrical installation, and all
        further activities subject to trade-specific regulation or permit requirements under the
        rules of the DGUV (the German Social Accident Insurance) and the MVStättVO (the Model
        Ordinance on Assembly Venues). The Contractor is not a licensed specialist for these areas.
        Responsibility and liability for these areas rests exclusively with the Organiser.
      </p>

      <h2 class="text-xl font-semibold">Section 3 Offer and conclusion of contract</h2>
      <p>
        The Contractor's quotations are non-binding and valid for 14 days from the date of issue,
        unless another period is stated. The contract is concluded by written order confirmation
        (email is sufficient) or by the commencement of performance. Oral side agreements require
        written confirmation.
      </p>

      <h2 class="text-xl font-semibold">Section 4 Remuneration and payment terms</h2>
      <p>
        The agreed remuneration is due per event day. Invoices are payable within 14 days of
        invoicing without deduction. For concert series or tours with multiple dates, the Contractor
        reserves the right to a deposit of 30% before the start of the tour.
      </p>
      <p>
        No value added tax is charged, under the small business exemption
        ("Kleinunternehmerregelung") of Section 19 of the German VAT Act (Umsatzsteuergesetz, UStG).
      </p>
      <p>
        In the event of late payment, the Contractor is entitled to charge default interest of 5%
        p.a. above the then-applicable base rate.
      </p>

      <h2 class="text-xl font-semibold">Section 5 Cancellation and cancellation fee</h2>
      <p>
        If the Organiser cancels or postpones one or more dates, the following cancellation fees
        apply, based on the agreed fee, per cancelled date:
      </p>
      <ul class="list-disc pl-6">
        <li>cancellation more than 60 days before the date: 25% of the fee</li>
        <li>cancellation 30–60 days before the date: 50% of the fee</li>
        <li>cancellation 14–29 days before the date: 75% of the fee</li>
        <li>cancellation less than 14 days before the date: 100% of the fee</li>
      </ul>
      <p>
        Costs already incurred (for example, equipment purchases or rentals demonstrably made for
        the booked production) are to be reimbursed in full, regardless of when the cancellation
        occurs. The Contractor is obliged to credit any income earned elsewhere for the cancelled
        period against this.
      </p>

      <h2 class="text-xl font-semibold">Section 6 Working conditions and working time</h2>
      <p>
        An event day (day rate) comprises up to 11 hours of working time, including one hour's
        break. Where this working time is exceeded, an additional day rate of the same amount is
        charged, unless otherwise agreed.
      </p>
      <p>
        The Contractor is entitled to interrupt or discontinue their work where the working
        conditions on site endanger their safety or the safety of the equipment, and the Organiser
        fails to remedy this despite being asked to. In this case, the Organiser has no claim to a
        refund of payments already made.
      </p>

      <h2 class="text-xl font-semibold">Section 7 Equipment – provision and liability</h2>
      <p>
        Equipment brought by the Contractor is provided in proper, functional condition. The
        Organiser is liable for damage to or loss of the equipment from the time of delivery until
        it is removed. This applies in particular to damage caused by improper handling by third
        parties (the audience, local staff), theft, and damage caused by natural events.
      </p>
      <p>
        Damaged or lost equipment will be invoiced at the current replacement price (new price). Any
        damage must be reported immediately, and at the latest by the time of load-out. The
        Contractor recommends that the Organiser take out event liability insurance.
      </p>
      <p>
        The Contractor is not liable for damage caused by a defective power supply (fluctuations,
        overvoltage) on the part of the Organiser or the venue.
      </p>

      <h2 class="text-xl font-semibold">Section 8 Organiser's obligations</h2>
      <p>The Organiser provides the following, free of charge:</p>
      <ul class="list-disc pl-6">
        <li>free and timely access to the venue from the agreed get-in time</li>
        <li>
          sufficiently dimensioned stage power (230V/16A Schuko, separate from lighting power) as
          well as FOH power
        </li>
        <li>
          an FOH position in the audience area (not in the production booth), at least 1 m wide
        </li>
        <li>a network connection (Cat 5e or higher) between FOH and the stage, where agreed</li>
        <li>
          local technical staff (lighting and house sound engineers, stagehands) in accordance with
          the stage plot
        </li>
        <li>
          obtaining all official permits, in particular noise-protection permits and exemptions for
          noise levels
        </li>
        <li>
          commissioning and bearing the cost of any sound-level measurements or noise-protection
          assessments
        </li>
        <li>complying with and monitoring the officially approved emission limits</li>
      </ul>
      <p>
        The Contractor will have regard to level limits on the Organiser's instruction, but bears no
        responsibility and no liability for regulatory measures resulting from exceeding approved
        limits.
      </p>
      <p>
        Any GEMA fees (the German music performance-rights organisation) and official permits are
        borne by the Organiser. For concert series, the venue specifications must be provided to the
        Contractor no later than 4 weeks before the respective event.
      </p>

      <h2 class="text-xl font-semibold">Section 9 Force majeure</h2>
      <p>
        Where the Contractor is unable to provide their service due to force majeure (natural
        disasters, official orders, power outages, pandemics or similar), they are released from the
        obligation to perform. Payments already made for the affected period will be refunded on a
        pro rata basis. Further claims for damages are excluded in this case.
      </p>

      <h2 class="text-xl font-semibold">Section 10 References and image rights</h2>
      <p>
        The Contractor is entitled to refer to the service provided for reference and advertising
        purposes. Unless the Organiser objects in writing, the Contractor may use their own
        recordings of the working environment (not of the artist or the show itself) for portfolio
        purposes.
      </p>

      <h2 class="text-xl font-semibold">Section 11 Data protection</h2>
      <p>
        In connection with the initiation and performance of contracts, the Contractor collects and
        stores the Organiser's personal data exclusively for the purpose of performing the contract.
        The data is not passed on to third parties unless required by law. On request, the data will
        be deleted once the business relationship has ended, to the extent no statutory retention
        periods apply.
      </p>

      <h2 class="text-xl font-semibold">Section 12 Final provisions</h2>
      <p>
        The law of the Federal Republic of Germany applies. The place of jurisdiction for all
        disputes is Aachen, to the extent legally permissible. Should individual provisions of these
        General Terms and Conditions be or become invalid, the validity of the remaining provisions
        remains unaffected. The invalid provision is replaced by the legally permissible provision
        that comes closest to its original purpose.
      </p>
      <p>There are no oral side agreements. Amendments and additions require written form.</p>
    </div>
  </GovueLegalPage>
</template>
