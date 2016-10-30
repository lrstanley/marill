angular.module('main', ['ngMaterial']);

angular.module('main').config(function($interpolateProvider) {
    // change angularjs interpoles to {[stuff]} rather than {{stuff}}
    $interpolateProvider.startSymbol('{[').endSymbol(']}');
});

angular.module('main').run(function($rootScope) {
    $rootScope.data = JSON.parse(document.getElementById('data').innerHTML);
    if (!$rootScope.data.Success) {
        // some kind of error occurred.
        console.log("ERRORS!");
    }
});

angular.module('main').controller('mainCtrl', function($scope, $rootScope) {
    console.log($rootScope.data);
});