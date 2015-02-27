$(document).ready(function() {

  function sideNav() {
    $('#nav-toggle').sidr({
      name: 'main-nav',
      source: '#navigation',
      side: 'right'
    });

    $('#sidr-id-nav-close').click(function () {
      $.sidr('close', 'main-nav');
    });

    $(window).touchwipe({
      wipeRight: function () {
        $.sidr('close', 'main-nav');
      },
      wipeLeft: function () {
        $.sidr('open', 'main-nav');
      },
      preventDefaultEvents: false
    });
  }

  function heroScroll(){
    var $header = $('header');
    var $hero = $('.hero');
    var height = $hero.css("height");

    $('section:nth-of-type(2)').css('margin-top', height);

    var size = window.matchMedia('all and (min-width: 650px)');

    if(size.matches) {
      $(window).scroll(function() {
        if( $(this).scrollTop() > height.replace('px', '') ) {
          $header.addClass('small');
        } else {
          $header.removeClass('small');
        }
      });
    }
  }

  sideNav();
  heroScroll();

});