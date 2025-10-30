export default function Loading() {
  return (
    <div className="min-h-screen bg-background">
      <div className="relative h-32 bg-background mb-8">
        <a href="https://hackclub.com/"></a>
      </div>
      <div className="max-w-4xl mx-auto px-6 py-12 mt-14 ">
        <header className="mb-16 text-start flex flex-col justify-start items-start ">
          <div className="h-12 bg-dark rounded mb-4 w-80 mx-auto animate-pulse"></div>
          <div className="h-6 bg-dark rounded w-96 mx-auto animate-pulse"></div>
        </header>
        <main>
          <div className="grid gap-6 md:gap-8">
            {Array.from({ length: 5 }).map((_, index) => (
              <div
                key={index}
                className="bg-dark rounded-xl shadow-sm border border-dark p-6 animate-pulse text-muted"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex-1 pr-4">
                    <div className="h-6 bg-slate rounded mb-2"></div>
                    <div className="h-4 bg-slate rounded w-3/4"></div>
                  </div>
                  <div className="h-6 w-20 bg-slate rounded-full"></div>
                </div>
                <div className="space-y-2 mb-6">
                  <div className="h-4 bg-slate rounded"></div>
                  <div className="h-4 bg-slate rounded w-5/6"></div>
                  <div className="h-4 bg-slate rounded w-4/6"></div>
                </div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-4">
                    <div className="h-8 w-20 bg-slate rounded-full"></div>
                    <div className="flex space-x-4">
                      <div className="h-6 w-16 bg-slate rounded-full"></div>
                      <div className="h-6 w-16 bg-slate rounded-full"></div>
                    </div>
                  </div>
                  <div className="h-8 w-24 bg-slate rounded-lg"></div>
                </div>
              </div>
            ))}
          </div>
        </main>
      </div>
    </div>
  );
}
